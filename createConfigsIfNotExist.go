package main

import "os"

func createConfigsIfNotExist() error {
	entries, err := os.ReadDir("templates")
	if err != nil {
		return err
	}
	for _, f := range entries {
		_, err := os.Stat(".private/" + f.Name())
		if err != nil {
			if os.IsNotExist(err) {
				err := copyFile("templates/"+f.Name(), ".private/"+f.Name())
				if err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}
	return nil
}
