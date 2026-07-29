package ffmpeg

import (
	"fmt"
	"os"
	"os/exec"
)

const (
	// originalVolume ducks the source audio so the translated voice sits on top of it.
	originalVolume = 0.2
	// translatedVolume passes the VOT track through at its original level.
	translatedVolume = 1.0
	// limiterCeiling caps the summed signal just below full scale. Without
	// normalize=0 amix would halve both inputs; without a limiter the sum
	// (up to originalVolume+translatedVolume) would hard-clip instead.
	limiterCeiling = 0.95
)

// AddAudio mixes the audio from inputAudio into inputVideo.
// It lowers the volume of the original video's audio and mixes strictly the new audio.
func AddAudio(inputVideo, inputAudio, outputFile string) error {
	// amix defaults to normalize=1, which divides every input by the number of
	// inputs — that alone made the translated voice 6 dB quieter than the source
	// mp3. normalize=0 keeps the levels set by the volume filters above, and
	// alimiter absorbs the resulting peaks so nothing clips.
	filter := fmt.Sprintf(
		"[0:a:0]volume=%.2f[a1];[1:a:0]volume=%.2f[a2];"+
			"[a1][a2]amix=inputs=2:normalize=0[mixed];"+
			"[mixed]alimiter=limit=%.2f:level=disabled[aout]",
		originalVolume, translatedVolume, limiterCeiling,
	)

	// Construct the ffmpeg command arguments for better readability
	// We use -y to overwrite the output file without asking
	args := []string{
		"-v", "error", // standard log level
		"-stats", // show progress
		"-i", inputVideo,
		"-i", inputAudio,
		"-filter_complex", filter,
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
