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

func getIDByAPI(ctx context.Context, cfg *config.API, claim bool, name string) (string, error) {
	idMeta, err := getDevboxIDMeta(name)
	if err != nil {
		return "", err
	}
	labels, err := getDevboxLabels()
	if err != nil {
		return "", err
	}
	req := &models.DevboxRegistration{
		IDMeta: idMeta,
		Labels: labels,
		Claim:  claim,
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

func getDevboxIDMeta(name string) (map[string]string, error) {
	if name == "" {
		h, err := os.Hostname()
		if err != nil {
			return nil, err
		}
		name = h
	}
	mid, err := system.GetMachineID()
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"name":       name,
		"machine-id": mid,
	}, nil
}

func getDevboxLabels() (map[string]string, error) {
	h, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"goos":   runtime.GOOS,
		"goarch": runtime.GOARCH,
		"host":   h,
	}, nil
}
