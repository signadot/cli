package planrunnergroup

import (
	"archive/tar"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-openapi/runtime"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/signadot/cli/internal/config"
	sdclient "github.com/signadot/go-sdk/client"
	planrunnergroups "github.com/signadot/go-sdk/client/plan_runner_groups"
	sdtransport "github.com/signadot/go-sdk/transport"
	"github.com/spf13/cobra"
)

func newImagePush(prg *config.PlanRunnerGroup) *cobra.Command {
	cfg := &config.PlanRunnerGroupImagePush{PlanRunnerGroup: prg}

	cmd := &cobra.Command{
		Use:   "push PRG_NAME IMAGE_REF",
		Short: "Push an image to a plan runner group",
		Long: `Pull an image from a registry (using local Docker credentials) and
upload it to the PRG's image cache. The PRG propagates the image to
all its pods automatically.

If --file is set, uploads an existing OCI archive instead of pulling.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return imagePush(cfg, cmd.OutOrStdout(), cmd.ErrOrStderr(), args[0], args[1])
		},
	}
	cfg.AddFlags(cmd)
	return cmd
}

func imagePush(cfg *config.PlanRunnerGroupImagePush, out, errOut io.Writer, prgName, imageRef string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}

	var archiveReader io.ReadCloser

	if cfg.File != "" {
		f, err := os.Open(cfg.File)
		if err != nil {
			return fmt.Errorf("open archive: %w", err)
		}
		archiveReader = f
		fmt.Fprintf(errOut, "Uploading %s as %s to %s\n", cfg.File, imageRef, prgName)
	} else {
		platform, err := resolvePlatform(cfg, prgName)
		if err != nil {
			return err
		}
		fmt.Fprintf(errOut, "Pulling %s (%s/%s)...\n", imageRef, platform.OS, platform.Architecture)

		ref, err := name.ParseReference(imageRef)
		if err != nil {
			return fmt.Errorf("invalid image reference: %w", err)
		}
		img, err := remote.Image(ref,
			remote.WithAuthFromKeychain(authn.DefaultKeychain),
			remote.WithPlatform(*platform))
		if err != nil {
			return fmt.Errorf("pull image: %w", err)
		}

		fmt.Fprintf(errOut, "Writing OCI archive...\n")
		archiveReader, err = writeOCIArchive(img, ref)
		if err != nil {
			return fmt.Errorf("write OCI archive: %w", err)
		}
	}
	defer archiveReader.Close()

	fmt.Fprintf(errOut, "Uploading to %s...\n", prgName)

	transportCfg := cfg.GetBaseTransport()
	transportCfg.OverrideProducers = true
	transportCfg.Producers = map[string]runtime.Producer{
		"application/x-tar": runtime.ByteStreamProducer(),
	}
	transportCfg.HTTPClient = &http.Client{Timeout: 30 * time.Minute}

	var result *sdtransport.UploadPRGImageResult
	err := cfg.APIClientWithCustomTransport(transportCfg,
		func(c *sdclient.SignadotAPI) error {
			var err error
			result, err = sdtransport.UploadPRGImage(
				c.Transport, cfg.Org, prgName, imageRef, archiveReader)
			return err
		})
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "digest: %s\n", result.Digest)
	fmt.Fprintf(out, "refs:   %v\n", result.Refs)
	fmt.Fprintf(out, "size:   %d\n", result.Size)
	return nil
}

func resolvePlatform(cfg *config.PlanRunnerGroupImagePush, prgName string) (*v1.Platform, error) {
	if cfg.Platform != "" {
		p, err := v1.ParsePlatform(cfg.Platform)
		if err != nil {
			return nil, fmt.Errorf("invalid platform %q: %w", cfg.Platform, err)
		}
		return p, nil
	}
	// Auto-detect from PRG pod arch.
	params := planrunnergroups.NewListPrgImagesParams().
		WithOrgName(cfg.Org).
		WithPlanRunnerGroupName(prgName)
	resp, err := cfg.Client.PlanRunnerGroups.ListPrgImages(params, nil)
	if err != nil {
		return nil, fmt.Errorf("auto-detect platform: %w", err)
	}
	var arch string
	if resp.Payload != nil {
		arch = resp.Payload.Arch
	}
	if arch == "" {
		arch = "amd64"
	}
	return &v1.Platform{OS: "linux", Architecture: arch}, nil
}

// writeOCIArchive writes an image as an OCI-layout tarball that
// UploadBytes can consume. Returns a ReadCloser over the tar; the
// caller must close it and the underlying temp file is cleaned up.
func writeOCIArchive(img v1.Image, ref name.Reference) (io.ReadCloser, error) {
	dir, err := os.MkdirTemp("", "signadot-oci-*")
	if err != nil {
		return nil, err
	}
	// Write OCI layout with the image appended to an empty index.
	idx := mutate.AppendManifests(empty.Index, mutate.IndexAddendum{Add: img})
	if _, err := layout.Write(dir, idx); err != nil {
		os.RemoveAll(dir)
		return nil, fmt.Errorf("write OCI layout: %w", err)
	}
	// Tar the layout directory into a temp file.
	tmp, err := os.CreateTemp("", "signadot-oci-*.tar")
	if err != nil {
		os.RemoveAll(dir)
		return nil, err
	}
	if err := tarDir(dir, tmp); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		os.RemoveAll(dir)
		return nil, fmt.Errorf("tar OCI layout: %w", err)
	}
	os.RemoveAll(dir)
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return nil, err
	}
	return &cleanupReadCloser{File: tmp}, nil
}

type cleanupReadCloser struct {
	*os.File
}

func (c *cleanupReadCloser) Close() error {
	err := c.File.Close()
	os.Remove(c.File.Name())
	return err
}

func tarDir(src string, w io.Writer) error {
	tw := tar.NewWriter(w)
	defer tw.Close()
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = rel
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()
			_, err = io.Copy(tw, f)
			return err
		}
		return nil
	})
}
