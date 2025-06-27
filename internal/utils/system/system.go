package system

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/panta/machineid"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Get signadot directory ($HOME/.signadot)
func GetSignadotDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
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
