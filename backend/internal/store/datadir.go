package store

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// EnsureDataDir membuat folder data milik perangkat ini. Folder yang baru dibuat diberi .gitignore
// agar akun dan token tidak ikut git bila folder berada di dalam repo. Folder yang sudah ada tidak
// disentuh, supaya .gitignore "*" tidak pernah menutupi source code.
func EnsureDataDir(dir string) error {
	if _, err := os.Stat(dir); err == nil {
		return nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*\n"), 0o600)
}
