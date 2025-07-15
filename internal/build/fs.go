package build

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// LockProject locks the build project. If the lockfile exists,
// LockProject returns (false, nil).
func (b *Build) LockProject() (ok bool, err error) {
	klarFolder := filepath.Join(b.ProjectDir, "/.klar")
	lockName := fmt.Sprintf("build-%s.klar-build", b.Target)
	err = os.MkdirAll(klarFolder, os.ModePerm)
	if err != nil {
		return false, err
	}
	_, err = os.Create(filepath.Join(klarFolder, lockName))
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}