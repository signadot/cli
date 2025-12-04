package devbox

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/utils/system"
	"github.com/signadot/go-sdk/client/devboxes"
	"github.com/signadot/go-sdk/models"
)

// GetSessionID tries to get the session id of devbox indicated by devboxID.  Upon success err is nil and
// if the devbox is connected, a non-empty string id is returned.  If the devbox is not connected, the
// empty string is returned.
func GetSessionID(ctx context.Context, apiConfig *config.API, devboxID string) (id string, err error) {
	params := devboxes.NewGetDevboxParams().
		WithContext(ctx).
		WithOrgName(apiConfig.Org).
		WithDevboxID(devboxID)

	resp, err := apiConfig.Client.Devboxes.GetDevbox(params)
	if err != nil {
		return "", err
	}
	switch resp.Code() {
	case http.StatusOK:
		session := resp.Payload.Status.Session
		if session == nil {
			return "", nil
		}
		return session.ID, nil
	default:
		return "", fmt.Errorf("error fetching devbox: status %d %s", resp.Code(), http.StatusText(resp.Code()))
	}
}

func GetID(ctx context.Context, apiConfig *config.API, claim bool, name string) (string, error) {
	file, err := IDFile()
	if err != nil {
		return "", err
	}
	d, err := os.ReadFile(file)
	id := string(d)
	if err != nil {
		if os.IsNotExist(err) {
			id, err := getIDByAPI(ctx, apiConfig, claim, name)
			if err != nil {
				return "", err
			}
			if err := os.WriteFile(file, []byte(id), 0600); err != nil {
				return "", err
			}
			return id, nil
		}
		return "", err
	}
	if name != "" {
		// get devbox, check name equals input.  if not,
		// call getIDByAPI with the new name to get the updated id
		// which will fall through to the claim code below
		params := devboxes.NewGetDevboxParams().
			WithContext(ctx).
			WithOrgName(apiConfig.Org).
			WithDevboxID(id)

		resp, err := apiConfig.Client.Devboxes.GetDevbox(params)
		if err != nil {
			return "", err
		}
		if resp.Code() == http.StatusOK && resp.Payload != nil {
			devboxName := ""
			if resp.Payload.Metadata != nil {
				devboxName = resp.Payload.Metadata["name"]
			}
			if devboxName != name {
				// Name doesn't match, get new ID with the updated name
				newID, err := getIDByAPI(ctx, apiConfig, claim, name)
				if err != nil {
					return "", err
				}
				return newID, nil
			}
		} else {
			return "", fmt.Errorf("unable to get devbox: status %d %s", resp.Code(), http.StatusText(resp.Code()))
		}
	}
	if !claim {
		return id, nil
	}
	params := devboxes.NewClaimDevboxParams().
		WithContext(ctx).
		WithOrgName(apiConfig.Org).
		WithDevboxID(id)

	_, err = apiConfig.Client.Devboxes.ClaimDevbox(params)
	if err != nil {
		return "", err
	}
	return id, nil
}

func IDFile() (string, error) {
	sdir, err := system.GetSignadotDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(sdir, ".devbox-id"), nil
}

// RegisterDevbox registers a devbox with the API and returns the devbox ID.
// If name is empty, it will use the hostname. If claim is true, it will also claim a session.
func RegisterDevbox(ctx context.Context, cfg *config.API, claim bool, name string) (string, error) {
	return getIDByAPI(ctx, cfg, claim, name)
}

func getIDByAPI(ctx context.Context, cfg *config.API, claim bool, name string) (string, error) {
	meta, err := getDevboxMeta(name)
	if err != nil {
		return "", err
	}
	req := &models.DevboxRegistration{
		Metadata: meta,
		Claim:    claim,
	}
	params := devboxes.NewRegisterDevboxParams().
		WithContext(ctx).
		WithOrgName(cfg.Org).
		WithData(req)
	regOK, regCreated, err := cfg.Client.Devboxes.RegisterDevbox(params)
	if err != nil {
		return "", err
	}
	var reg *models.Devbox
	if regOK != nil {
		reg = regOK.Payload
	} else if regCreated != nil {
		reg = regCreated.Payload
	}
	return reg.ID, nil
}

func getDevboxMeta(name string) (map[string]string, error) {
	host, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	if name == "" {
		name = host
	}
	mid, err := system.GetMachineID()
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"name":       name,
		"machine-id": mid,
		"goos":       runtime.GOOS,
		"goarch":     runtime.GOARCH,
		"host":       host,
	}, nil
}
