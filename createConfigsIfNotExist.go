package main

import (
	"os"
	"path/filepath"
)

func createConfigsIfNotExist() (bool, error) {
	const (
		templatesDir = "templates"
		targetDir    = ".private"
	)

	entries, err := os.ReadDir(templatesDir)
	if err != nil {
		return false, err
	}

	// Убедимся, что целевая директория существует
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return false, err
	}

	var created bool
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		targetPath := filepath.Join(targetDir, entry.Name())
		if _, err := os.Stat(targetPath); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return false, err
		}

		srcPath := filepath.Join(templatesDir, entry.Name())
		if err := copyFile(srcPath, targetPath); err != nil {
			return false, err
		}
		created = true
	}

	return created, nil
}
