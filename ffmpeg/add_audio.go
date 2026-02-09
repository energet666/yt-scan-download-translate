package ffmpeg

import (
	"fmt"
	"os"
	"os/exec"
)

// AddAudio mixes the audio from inputAudio into inputVideo.
// It lowers the volume of the original video's audio and mixes strictly the new audio.
func AddAudio(inputVideo, inputAudio, outputFile string) error {
	// Construct the ffmpeg command arguments for better readability
	// We use -y to overwrite the output file without asking
	args := []string{
		"-v", "error", // standard log level
		"-stats", // show progress
		"-i", inputVideo,
		"-i", inputAudio,
		"-filter_complex", "[0:a:0]volume=0.2[a1];[1:a:0]volume=1.0[a2];[a1][a2]amix=inputs=2[aout]",
		"-map", "0:v", // copy video stream from first input
		"-map", "[aout]", // use the mixed audio stream
		"-c:v", "copy", // copy video codec (no re-encode)
		"-c:a", "aac", // re-encode audio to aac
		outputFile,
		"-y",
	}

	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add audio: %w", err)
	}

	return nil
}
