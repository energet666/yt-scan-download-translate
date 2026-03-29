package votclilive

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

const downloadTimeout = 5 * time.Minute

func Download(url string, path string, filename string, voiceStyle string) error {
	ctx, cancel := context.WithTimeout(context.Background(), downloadTimeout)
	defer cancel()

	args := []string{
		"--voice-style=" + voiceStyle,
		"--output=" + path,
		"--output-file=" + filename,
		url,
	}
	cmd := exec.CommandContext(ctx, "vot-cli-live", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("vot-cli-live timed out after %v", downloadTimeout)
	}
	return err
}
