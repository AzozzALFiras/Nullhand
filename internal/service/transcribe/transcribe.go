// Package transcribe converts audio files (typically Telegram voice notes
// in Opus .ogg format) to text using whisper.cpp.
//
// The package shells out to whisper-cli (the new entrypoint) or `whisper`
// (the older one), and to ffmpeg for the .ogg → .wav conversion. Both must
// be installed; on first failure we return a descriptive error so the bot
// can suggest install commands.
package transcribe

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ErrWhisperMissing is returned when neither whisper-cli nor whisper is on PATH.
var ErrWhisperMissing = errors.New(
	"whisper.cpp not installed.\n" +
		"  Linux:  https://github.com/ggerganov/whisper.cpp (build whisper-cli)\n" +
		"  macOS:  brew install whisper-cpp",
)

// ErrFfmpegMissing is returned when ffmpeg is not on PATH.
var ErrFfmpegMissing = errors.New(
	"ffmpeg not installed (needed to convert Telegram .ogg → .wav).\n" +
		"  Linux:  sudo apt install ffmpeg\n" +
		"  macOS:  brew install ffmpeg",
)

// Options controls a single transcription call.
type Options struct {
	// Language hint. Use "auto" to let whisper detect (slower). Most users
	// want "ar" or "en"; bilingual users can pass "ar" — whisper-cli's Arabic
	// model handles English code-switches inside Arabic speech well enough.
	Language string

	// ModelPath optionally overrides whisper-cli's default model lookup with
	// an explicit path to a ggml-*.bin file. Empty = let whisper-cli pick.
	ModelPath string
}

// Transcribe converts the given .ogg or .wav audio bytes to text.
//
// Pipeline: write bytes to temp file → if .ogg, ffmpeg-convert to .wav → call
// whisper-cli with -otxt and read the resulting .txt. All temp files are
// removed before returning.
func Transcribe(audioBytes []byte, sourceMime string, opts Options) (string, error) {
	whisperBin, err := findWhisper()
	if err != nil {
		return "", err
	}

	// 1. Write raw audio to temp file with a sane extension.
	ext := ".ogg"
	switch {
	case strings.Contains(sourceMime, "wav"), strings.Contains(sourceMime, "wave"):
		ext = ".wav"
	case strings.Contains(sourceMime, "mp3"):
		ext = ".mp3"
	case strings.Contains(sourceMime, "m4a"), strings.Contains(sourceMime, "mp4"):
		ext = ".m4a"
	case strings.Contains(sourceMime, "ogg"):
		ext = ".ogg"
	}
	srcFile, err := os.CreateTemp("", "nullhand-voice-*"+ext)
	if err != nil {
		return "", fmt.Errorf("transcribe: temp file: %w", err)
	}
	srcPath := srcFile.Name()
	defer os.Remove(srcPath)
	if _, err := srcFile.Write(audioBytes); err != nil {
		srcFile.Close()
		return "", fmt.Errorf("transcribe: write audio: %w", err)
	}
	srcFile.Close()

	// 2. Convert to WAV if not already (whisper-cli accepts WAV directly).
	wavPath := srcPath
	if ext != ".wav" {
		if _, lookErr := exec.LookPath("ffmpeg"); lookErr != nil {
			return "", ErrFfmpegMissing
		}
		wavPath = strings.TrimSuffix(srcPath, ext) + ".wav"
		defer os.Remove(wavPath)
		// 16 kHz mono is what ggml whisper expects.
		convCmd := exec.Command("ffmpeg", "-y", "-i", srcPath,
			"-ar", "16000", "-ac", "1", "-c:a", "pcm_s16le", wavPath)
		out, err := convCmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("transcribe: ffmpeg failed: %w — %s", err, lastNonEmptyLine(string(out)))
		}
	}

	// 3. Run whisper-cli. Output goes to <wavPath>.txt by default with -otxt.
	args := []string{wavPath, "-otxt", "-nt"} // -nt = no timestamps in stdout
	if opts.Language != "" {
		args = append(args, "-l", opts.Language)
	}
	if opts.ModelPath != "" {
		args = append(args, "-m", opts.ModelPath)
	}
	whisperCmd := exec.Command(whisperBin, args...)
	out, err := whisperCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("transcribe: whisper failed: %w — %s", err, lastNonEmptyLine(string(out)))
	}

	// 4. Read the resulting <wavPath>.txt — whisper-cli writes it next to the input.
	txtPath := wavPath + ".txt"
	defer os.Remove(txtPath)
	data, err := os.ReadFile(txtPath)
	if err != nil {
		// Fallback: try parsing stdout if whisper printed the transcript.
		text := extractTranscriptFromStdout(string(out))
		if text != "" {
			return text, nil
		}
		return "", fmt.Errorf("transcribe: read output: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

// findWhisper looks up the whisper.cpp executable in PATH. Tries "whisper-cli"
// (newer name) first, then "whisper" and "main" (older bundled names).
// Returns the absolute path or ErrWhisperMissing.
func findWhisper() (string, error) {
	for _, name := range []string{"whisper-cli", "whisper-cpp", "whisper", "main"} {
		if p, err := exec.LookPath(name); err == nil {
			return p, nil
		}
	}
	return "", ErrWhisperMissing
}

// IsAvailable reports whether whisper.cpp + ffmpeg are both installed and
// callable. Used by /health and the bot startup probe.
func IsAvailable() (bool, string) {
	whisper, err := findWhisper()
	if err != nil {
		return false, "whisper.cpp missing"
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return false, "ffmpeg missing"
	}
	return true, filepath.Base(whisper)
}

// extractTranscriptFromStdout pulls a likely-transcript line from whisper-cli
// stdout for cases where -otxt didn't produce a file. Heuristic: pick the
// longest non-status line that doesn't start with "whisper_" or "[".
func extractTranscriptFromStdout(s string) string {
	var best string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "whisper_") || strings.HasPrefix(line, "[") || strings.HasPrefix(line, "system_info") {
			continue
		}
		if len(line) > len(best) {
			best = line
		}
	}
	return best
}

// lastNonEmptyLine returns the last non-empty line of s, useful for
// summarising stderr from external tools in error messages.
func lastNonEmptyLine(s string) string {
	lines := strings.Split(s, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		l := strings.TrimSpace(lines[i])
		if l != "" {
			return l
		}
	}
	return ""
}
