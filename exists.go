package pelican

import (
	"os"
)

func FileExists(name string) bool {
	fi, err := os.Stat(name)
	if err != nil {
		return false
	}
	if fi.IsDir() {
		return false
	}
	return true
}

func FileExistsLen(name string) (bool, int64) {
	fi, err := os.Stat(name)
	if err != nil {
		return false, 0
	}
	if fi.IsDir() {
		return false, 0
	}
	return true, fi.Size()
}

func DirExists(name string) bool {
	fi, err := os.Stat(name)
	if err != nil {
		return false
	}
	if fi.IsDir() {
		return true
	}
	return false
}
