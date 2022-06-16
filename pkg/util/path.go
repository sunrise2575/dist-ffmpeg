package util

import (
	"os"

	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

func PathSplit(path string) (string, string, string) {
	path, e := filepath.Abs(path)
	if e != nil {
		logrus.WithFields(
			logrus.Fields{
				"filepath_target": path,
				"error":           e,
				"where":           GetCurrentFunctionInfo(),
			}).Fatalf("filepath.Abs() failed")
	}
	dir, right := filepath.Split(path)
	ext := filepath.Ext(right)
	name := strings.TrimSuffix(right, ext)
	return dir, name, ext
}

func PathJoin(dir, name, ext string) string {
	return filepath.Join(dir, name+ext)
}

func PathSanitize(path string) string {
	path, e := filepath.Abs(path)
	if e != nil {
		logrus.WithFields(
			logrus.Fields{
				"filepath_target": path,
				"error":           e,
				"where":           GetCurrentFunctionInfo(),
			}).Fatalf("filepath.Abs() failed")
	}

	return path
}

// IsFile checks that the path is a file
func PathIsFile(path string) bool {
	fileStat, e := os.Stat(path)

	if os.IsNotExist(e) || fileStat.IsDir() {
		return false
	}

	return true
}

// IsDir checks that the path is a directory
func PathIsDir(path string) bool {
	fileStat, e := os.Stat(path)

	if os.IsNotExist(e) || !fileStat.IsDir() {
		return false
	}

	return true
}
