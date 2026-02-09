package votclilive

import (
	"os"
	"os/exec"
)

func Download(url string, path string, filename string) error {
	args := []string{
		"--output=" + path,
		"--output-file=" + filename,
		url,
	}
	cmd := exec.Command("vot-cli-live", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
