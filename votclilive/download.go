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

	// Capture both stdout and stderr while still printing to console
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)

	err := cmd.Run()

	// Check for "no speech" message in both stdout and stderr
	combined := stdoutBuf.String() + stderrBuf.String()
	if strings.Contains(combined, "нет речи") {
		return ErrNoSpeech
	}

	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("vot-cli-live timed out after %v", downloadTimeout)
	}
	return err
}
