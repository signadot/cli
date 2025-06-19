package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/signadot/cli/internal/utils/system"
)

var (
	gcPathsFile *os.File
	mu          sync.Mutex
)

func RegisterPathForGC(p string) error {
	mu.Lock()
	defer mu.Unlock()
	f, err := getGCPathsFile()
	if err != nil {
		return fmt.Errorf("cannot get registered-files: %w", err)
	}
	d, err := json.Marshal(p)
	if err != nil {
		return err
	}
	_, err = f.Write(d)
	if err != nil {
		return err
	}
	_, err = f.Write([]byte("\n"))
	return err
}

func GCPaths() error {
	mu.Lock()
	defer mu.Unlock()
	_, err := getGCPathsFile()
	if err != nil {
		return err
	}
	if gcPathsFile == nil {
		return nil
	}
	defer func() {
		gcPathsFile.Close()
		os.RemoveAll(gcPathsFile.Name())
		gcPathsFile = nil
	}()
	_, err = gcPathsFile.Seek(0, 0)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(gcPathsFile)
	var errs error
	cache := map[string]bool{}
	for {
		p := ""
		err := dec.Decode(&p)
		if errors.Is(err, io.EOF) {
			return errs
		}
		if err != nil {
			return errors.Join(errs, err)
		}
		if !cache[p] {
			errs = errors.Join(errs, os.RemoveAll(p))
			cache[p] = true
		}
	}
	return errs
}

func getGCPathsPath() (string, error) {
	sd, err := system.GetSignadotDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(sd, "gc-files.jsons"), nil
}

func getGCPathsFile() (*os.File, error) {
	if gcPathsFile != nil {
		return gcPathsFile, nil
	}
	p, err := getGCPathsPath()
	if err != nil {
		return nil, err
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	gcPathsFile = f
	return f, nil
}
