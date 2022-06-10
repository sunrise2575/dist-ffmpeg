package fsys

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

func Split(path string) (string, string, string) {
	path, e := filepath.Abs(path)
	if e != nil {
		log.Panicf("[PANIC] filepath.Abs(%v) failed, error: %v", path, e)
	}
	dir, right := filepath.Split(path)
	ext := filepath.Ext(right)
	name := strings.TrimSuffix(right, ext)
	return dir, name, ext
}

func Join(dir, name, ext string) string {
	return filepath.Join(dir, name+ext)
}

func Sanitize(path string) string {
	path, e := filepath.Abs(path)
	if e != nil {
		log.Panicf("[PANIC] filepath.Abs(%v) failed, error: %v", path, e)
	}

	return path
}

// IsFile checks that the path is a file
func IsFile(path string) bool {
	fileStat, e := os.Stat(path)

	if os.IsNotExist(e) || fileStat.IsDir() {
		return false
	}

	return true
}

// IsDir checks that the path is a directory
func IsDir(path string) bool {
	fileStat, e := os.Stat(path)

	if os.IsNotExist(e) || !fileStat.IsDir() {
		return false
	}

	return true
}
