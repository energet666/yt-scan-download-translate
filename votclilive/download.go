package votclilive

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

const downloadTimeout = 5 * time.Minute

// ErrNoSpeech is returned when the video has no speech to translate.
var ErrNoSpeech = errors.New("video has no speech to translate")

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

	// Capture stderr while still printing it to console
	var stderrBuf bytes.Buffer
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)

	err := cmd.Run()

	// Check for "no speech" message regardless of exit code
	if strings.Contains(stderrBuf.String(), "нет речи") {
		return ErrNoSpeech
	}

	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("vot-cli-live timed out after %v", downloadTimeout)
	}
	return err
}
