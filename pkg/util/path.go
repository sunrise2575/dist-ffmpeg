package util

import (
	"fmt"
	"io"
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
				"path_target": path,
				"error":       e,
				"where":       GetCurrentFunctionInfo(),
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
				"path_target": path,
				"error":       e,
				"where":       GetCurrentFunctionInfo(),
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

func PathExists(name string) bool {
	_, err := os.Stat(name)
	return !os.IsNotExist(err)
}

func PathMove(sourcePath, destPath string) error {
	sourceAbs, err := filepath.Abs(sourcePath)
	if err != nil {
		return err
	}
	destAbs, err := filepath.Abs(destPath)
	if err != nil {
		return err
	}
	if sourceAbs == destAbs {
		return nil
	}
	inputFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}

	destDir := filepath.Dir(destPath)
	if !PathExists(destDir) {
		err = os.MkdirAll(destDir, 0644)
		if err != nil {
			return err
		}
	}
	outputFile, err := os.Create(destPath)
	if err != nil {
		inputFile.Close()
		return err
	}

	_, err = io.Copy(outputFile, inputFile)
	inputFile.Close()
	outputFile.Close()
	if err != nil {
		if errRem := os.Remove(destPath); errRem != nil {
			return fmt.Errorf(
				"unable to os.Remove error: %s after io.Copy error: %s",
				errRem,
				err,
			)
		}
		return err
	}

	return os.Remove(sourcePath)
}
