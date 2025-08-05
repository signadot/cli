package system

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"

	"github.com/panta/machineid"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Get signadot directory ($HOME/.signadot)
func GetSignadotDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("error getting user home dir: %w", err)
	}
	return filepath.Join(homeDir, ".signadot"), nil
}

// Create directory if not exist
func CreateDirIfNotExist(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	}
	return nil
}

func GetRollingLogWriter(logDirPath, logName string, uid, gid int) (io.Writer, string, error) {
	// create directory if not exists.
	CreateDirIfNotExist(logDirPath)
	logPath := path.Join(logDirPath, logName)

	// create empty file if doesn't exist
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		file, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return nil, "", err
		}
		file.Chown(uid, gid)
		file.Close()
	}

	// return the rotating log writer
	return &lumberjack.Logger{
		Filename:   logPath,
		MaxBackups: 20, // files
		MaxSize:    50, // megabytes
		MaxAge:     28, // days
	}, logPath, nil
}

func GetMachineID() (string, error) {
	machineID, err := machineid.ProtectedID("signadotCLI")
	if err != nil {
		return "", fmt.Errorf("couldn't read machine-id, %v", err)
	}
	return machineID[:63], nil
}

// OpenBrowser opens the specified URL in the user's default browser
func OpenBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("unable to open url %q: %w", url, err)
	}
	return nil
}

func GetSandboxesDir() (string, error) {
	sdDir, err := GetSignadotDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(sdDir, "sandboxes"), nil
}

func GetSandboxLocalFilesBaseDir(sbName string) (string, error) {
	sbxsDir, err := GetSandboxesDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(sbxsDir, sbName, "local", "files"), nil
}
