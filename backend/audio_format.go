package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// resolveAudioContentType maps OpenAI response_format values to MIME types and filenames.
func resolveAudioContentType(format string) (string, string) {
	switch strings.ToLower(format) {
	case "mp3":
		return "audio/mpeg", "mimotts.mp3"
	case "opus":
		return "audio/ogg", "mimotts.opus"
	case "aac":
		return "audio/aac", "mimotts.aac"
	case "flac":
		return "audio/flac", "mimotts.flac"
	case "pcm":
		return "audio/pcm", "mimotts.pcm"
	default:
		return "audio/wav", "mimotts.wav"
	}
}

// isValidResponseFormat reports whether format is one of the OpenAI-standard
// speech response formats: mp3, opus, aac, flac, wav, pcm.
func isValidResponseFormat(format string) bool {
	switch strings.ToLower(format) {
	case "mp3", "opus", "aac", "flac", "wav", "pcm":
		return true
	default:
		return false
	}
}

// convertAudio converts WAV bytes to the target format and/or applies speed
// using ffmpeg. Returns the original WAV bytes if ffmpeg is unavailable or
// conversion fails (safe degradation — existing behavior is preserved).
func convertAudio(wavBytes []byte, targetFormat string, speed float64) []byte {
	needConvert := targetFormat != "" && !strings.EqualFold(targetFormat, "wav")
	needSpeed := speed > 0

	if !needConvert && !needSpeed {
		return wavBytes
	}

	// Build ffmpeg args. Always decode from raw WAV on stdin, output to stdout.
	args := []string{"-i", "pipe:0"}

	if needSpeed {
		if speed < 0.25 {
			speed = 0.25
		}
		if speed > 4.0 {
			speed = 4.0
		}
		// atempo supports 0.5–2.0 in a single filter; chain for higher ranges.
		args = append(args, "-filter:a", atempoFilter(speed))
	}

	var outExt string
	if needConvert {
		outExt = strings.ToLower(targetFormat)
	} else {
		outExt = "wav"
	}

	switch outExt {
	case "mp3":
		args = append(args, "-f", "mp3")
	case "opus":
		args = append(args, "-f", "ogg", "-c:a", "libopus")
	case "aac":
		args = append(args, "-f", "adts", "-c:a", "aac")
	case "flac":
		args = append(args, "-f", "flac")
	case "pcm":
		args = append(args, "-f", "s16le", "-c:a", "pcm_s16le")
	default:
		args = append(args, "-f", "wav")
	}

	args = append(args, "-y", "pipe:1")

	cmd := exec.Command("ffmpeg", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdin = bytes.NewReader(wavBytes)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "ffmpeg conversion failed: %v (stderr: %s)\n", err, truncateBytes(stderr.Bytes(), 200))
		return wavBytes
	}
	if stdout.Len() == 0 {
		return wavBytes
	}
	return stdout.Bytes()
}

// atempoFilter builds an ffmpeg atempo filter chain for arbitrary speed (0.25–4.0).
func atempoFilter(speed float64) string {
	if speed >= 0.5 && speed <= 2.0 {
		return "atempo=" + strconv.FormatFloat(speed, 'f', -1, 64)
	}
	// Chain: multiply to stay within 0.5–2.0 per stage.
	var filters []string
	remaining := speed
	// Faster than 2x: apply atempo=2.0 repeatedly.
	for remaining > 2.0 {
		filters = append(filters, "atempo=2.0")
		remaining /= 2.0
	}
	// Slower than 0.5x: apply atempo=0.5 repeatedly.
	for remaining < 0.5 {
		filters = append(filters, "atempo=0.5")
		remaining /= 0.5
	}
	if remaining != 1.0 {
		filters = append(filters, "atempo="+strconv.FormatFloat(remaining, 'f', -1, 64))
	}
	return strings.Join(filters, ",")
}

func truncateBytes(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "..."
}

func init() {
	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARNING: ffmpeg not found, audio format conversion will be disabled\n")
	}
}