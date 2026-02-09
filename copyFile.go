package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// copyFile copies a file from src to dst.
// It preserves the file mode (permissions).
func copyFile(src, dst string) (err error) {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Получаем информацию о файле через дескриптор (быстрее и надёжнее)
	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return err
	}

	// Проверяем, что это обычный файл
	if !sourceInfo.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	// Создаём директорию назначения, если её нет
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}

	// Используем именованный возврат для корректной обработки ошибки Close
	defer func() {
		cerr := destFile.Close()
		if err == nil {
			err = cerr
		}
	}()

	if _, err = io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	// Сбрасываем буферы на диск
	if err = destFile.Sync(); err != nil {
		return err
	}

	// Копируем права доступа
	if err = destFile.Chmod(sourceInfo.Mode()); err != nil {
		return err
	}

	return nil
}
