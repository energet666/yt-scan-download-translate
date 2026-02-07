package main

import "os"

func createConfigsIfNotExist() (bool, error) {
	entries, err := os.ReadDir("templates")
	if err != nil {
		return false, err
	}
	created := false
	for _, f := range entries {
		_, err := os.Stat(".private/" + f.Name())
		if err != nil {
			if os.IsNotExist(err) {
				err := copyFile("templates/"+f.Name(), ".private/"+f.Name())
				if err != nil {
					return false, err
				}
				created = true
			} else {
				return false, err
			}
		}
	}
	return created, nil
}
