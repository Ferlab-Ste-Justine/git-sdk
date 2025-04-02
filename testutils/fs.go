package testutils

import (
	"io/fs"
	"io/ioutil"
	"path/filepath"
	"strings"
)

type DirectoryContent map[string]string

func (cont DirectoryContent) Equals(other DirectoryContent) bool {
	for key, val := range cont {
		if otherVal, ok := other[key]; ok {
			if val != otherVal {
				return false
			}
		} else {
			return false
		}
	}

	for key, otherVal := range other {
		if val, ok := cont[key]; ok {
			if val != otherVal {
				return false
			}
		} else {
			return false
		}
	}

	return true
}

func GetDirectoryContent(rootPath string, skipPrefix string) (DirectoryContent, error) {
	dirCont := make(map[string]string)

	err := filepath.WalkDir(rootPath, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !entry.IsDir() {
			relPath, relPathErr := filepath.Rel(rootPath, path)
			if err != nil {
				return relPathErr
			}

			if skipPrefix != "" && strings.HasPrefix(relPath, skipPrefix) {
				return nil
			}

			content, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			dirCont[filepath.ToSlash(relPath)] = string(content)
		}

		return nil
	})

	return DirectoryContent(dirCont), err
}