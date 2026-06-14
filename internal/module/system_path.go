package module

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// KlarDataDir returns the directory where Klar data is stored. The data
// directory is where installed packages are stored. If the $KLAR_DIR
// environment variable is set, it is used as the data directory. Otherwise,
// the data directory is located in the user's system data directory
// (e.g. ~/.local/share/klar on Linux).
func KlarDataDir() (string, error) {
	if klarDir := os.Getenv("KLAR_DIR"); klarDir != "" {
		return klarDir, nil
	}
	// Linux: ~/.local/share
	// See https://specifications.freedesktop.org/basedir/latest/
	if localShare := os.Getenv("XDG_DATA_HOME"); localShare != "" {
		return filepath.Join(localShare, "klar"), nil
	}
	if runtime.GOOS == "windows" {
		localAppData := os.Getenv("LocalAppData")
		if localAppData == "" {
			return "", errors.New("%LocalAppData% variable is not defined")
		}
		return filepath.Join(localAppData, "Klar"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if runtime.GOOS == "darwin" || runtime.GOOS == "ios" {
		return filepath.Join(home, "Library", "Application Support", "Klar"), nil
	}
	// Linux or any other OS
	return filepath.Join(home, ".local", "share", "klar"), nil
}

// KlarStdDir returns the directory where Klar standard library source code
// is stored. If a $KLAR_STD environment variable is set, it is returned.
// Otherwise, if the current executable is located in the user's home directory,
// the data directory is located in the user's system data directory
// (e.g. ~/.local/share/klar on Linux). Otherwise, the data directory is located
// in the system root. (e.g. /usr/share/klar on Linux).
func KlarStdDir() (string, error) {
	if klarStd := os.Getenv("KLAR_STD"); klarStd != "" {
		return klarStd, nil
	}
	execPath, err := os.Executable()
	if err != nil {
		return "", err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	var isInHome bool
	if _, err := filepath.Rel(home, execPath); err == nil {
		isInHome = true
	}
	if isInHome {
		// User installation of Klar
		klarDataDir, err := KlarDataDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(klarDataDir, "std"), nil
	}
	// Klar installed system-wide
	switch runtime.GOOS {
	default:
		return filepath.Join("/usr", "share", "klar", "std"), nil
	case "darwin", "ios":
		return filepath.Join("/Library", "Application Support", "Klar", "std"), nil
	case "windows":
		programData := os.Getenv("ProgramData")
		if programData == "" {
			return "", errors.New("%ProgramData% variable is not defined")
		}
		return filepath.Join(programData, "Klar", "std"), nil
	}
}

func KlarCacheDir() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	switch runtime.GOOS {
	case "windows": // Local AppData, same as [KlarDataDir]
		return filepath.Join(dir, "Klar", "cache"), nil
	case "darwin", "ios":
		return filepath.Join(dir, "Klar"), nil
	default:
		return filepath.Join(dir, "klar"), nil
	}
}

func KlarConfigDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	switch runtime.GOOS {
	case "windows": // Roaming AppData
		return filepath.Join(dir, "Klar"), nil
	case "darwin", "ios":
		return filepath.Join(dir, "Klar"), nil
	default:
		return filepath.Join(dir, "klar"), nil
	}
}

type systemDirs struct {
	Data   string
	Cache  string
	Config string
	Std    string
}

var SystemDirs *systemDirs

func LoadSystemDirs() (err error) {
	var (
		klarData, err1   = KlarDataDir()
		klarCache, err2  = KlarCacheDir()
		klarConfig, err3 = KlarConfigDir()
		klarStd, err4    = KlarStdDir()
	)
	if err = cmp.Or(err1, err2, err3, err4); err != nil {
		return fmt.Errorf("failed to load Klar directories: %w", err)
	}
	SystemDirs = &systemDirs{
		Data:   klarData,
		Cache:  klarCache,
		Config: klarConfig,
		Std:    klarStd,
	}
	return nil
}
