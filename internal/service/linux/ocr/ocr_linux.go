//go:build linux

package ocr

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	screensvc "github.com/AzozzALFiras/Nullhand/internal/service/linux/screen"
)

// TextBox is a recognized text region on screen with its bounding box.
// Coordinates are in screen pixels (top-left origin).
type TextBox struct {
	Text   string
	X, Y   int // top-left
	W, H   int // width/height
	CenterX, CenterY int
}

// ErrNotInstalled is returned when tesseract is not found on the system.
var ErrNotInstalled = errors.New("tesseract is not installed — run: sudo apt install tesseract-ocr")

// ReadScreen captures the current screen and extracts visible text via
// Tesseract OCR. Returns the extracted text (trimmed), or ErrNotInstalled if
// tesseract is not available, or another error if capture/OCR fails.
// Extracted text is truncated to 4096 characters (Telegram message limit).
func ReadScreen() (string, error) {
	// 1. Take screenshot
	data, err := screensvc.Capture()
	if err != nil {
		return "", fmt.Errorf("ocr: screenshot: %w", err)
	}

	// 2. Write to temp file
	tmp, err := os.CreateTemp("", "nullhand-ocr-*.png")
	if err != nil {
		return "", fmt.Errorf("ocr: create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return "", fmt.Errorf("ocr: write temp file: %w", err)
	}
	tmp.Close()

	// 3. Run tesseract
	cmd := exec.Command("tesseract", tmpPath, "stdout", "-l", "eng")
	out, err := cmd.Output()
	if err != nil {
		// Check if tesseract is missing entirely
		if _, lookErr := exec.LookPath("tesseract"); lookErr != nil {
			return "", ErrNotInstalled
		}
		// Tesseract exits non-zero on some images even when it produces output.
		// Return whatever stdout we got; if empty the caller handles it.
		text := strings.TrimSpace(string(out))
		if text == "" {
			return "", fmt.Errorf("ocr: tesseract failed: %w", err)
		}
		return truncate(text, 4096), nil
	}

	text := strings.TrimSpace(string(out))
	return truncate(text, 4096), nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

// hocrBboxRe extracts bounding box and text from an hocr ocrx_word span:
//   <span class='ocrx_word' id='word_1_1' title='bbox 100 200 300 240; ...'>Hello</span>
var hocrBboxRe = regexp.MustCompile(
	`(?is)<span\s+class=['"]ocrx_word['"][^>]*title=['"][^'"]*bbox\s+(\d+)\s+(\d+)\s+(\d+)\s+(\d+)[^'"]*['"][^>]*>([^<]*)</span>`)

// hocrLineRe extracts a line bbox span (used as a fallback if word-level fails).
var hocrLineRe = regexp.MustCompile(
	`(?is)<span\s+class=['"]ocr_line['"][^>]*title=['"]bbox\s+(\d+)\s+(\d+)\s+(\d+)\s+(\d+)[^'"]*['"][^>]*>(.*?)</span>`)

// stripTags strips HTML tags from a fragment.
var hocrTagRe = regexp.MustCompile(`<[^>]+>`)

// LocateText takes a screenshot, runs tesseract HOCR, and returns the bounding
// box of the first text region matching needle (case-insensitive substring).
// Returns found=false if not found. Returns ErrNotInstalled if tesseract missing.
func LocateText(needle string) (TextBox, bool, error) {
	if needle == "" {
		return TextBox{}, false, fmt.Errorf("LocateText: needle is required")
	}
	boxes, err := scanScreenHOCR()
	if err != nil {
		return TextBox{}, false, err
	}
	target := strings.ToLower(strings.TrimSpace(needle))

	// Pass 1: exact-ish word/phrase match in any single box.
	for _, b := range boxes {
		if strings.Contains(strings.ToLower(b.Text), target) {
			return b, true, nil
		}
	}

	// Pass 2: token sequence — needle words appear consecutively across boxes
	// on the same row. Useful for multi-word phrases tesseract split into words.
	parts := strings.Fields(target)
	if len(parts) >= 2 {
		for i := 0; i < len(boxes); i++ {
			if !strings.Contains(strings.ToLower(boxes[i].Text), parts[0]) {
				continue
			}
			row := boxes[i].Y + boxes[i].H/2
			matched := []TextBox{boxes[i]}
			j := i + 1
			for k := 1; k < len(parts) && j < len(boxes); j++ {
				bRow := boxes[j].Y + boxes[j].H/2
				if abs(bRow-row) > boxes[i].H {
					continue
				}
				if strings.Contains(strings.ToLower(boxes[j].Text), parts[k]) {
					matched = append(matched, boxes[j])
					k++
					if k >= len(parts) {
						return mergeBoxes(matched), true, nil
					}
				}
			}
		}
	}

	return TextBox{}, false, nil
}

// LocateAllText returns all OCR text boxes whose text contains needle.
func LocateAllText(needle string) ([]TextBox, error) {
	if needle == "" {
		return nil, fmt.Errorf("LocateAllText: needle is required")
	}
	boxes, err := scanScreenHOCR()
	if err != nil {
		return nil, err
	}
	target := strings.ToLower(strings.TrimSpace(needle))
	var out []TextBox
	for _, b := range boxes {
		if strings.Contains(strings.ToLower(b.Text), target) {
			out = append(out, b)
		}
	}
	return out, nil
}

// WaitForText polls every 400ms until needle is visible on screen via OCR,
// or timeoutMs expires. Returns the matched bounding box on success.
func WaitForText(needle string, timeoutMs int) (TextBox, error) {
	if timeoutMs <= 0 {
		timeoutMs = 5000
	}
	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	for {
		box, found, err := LocateText(needle)
		if err == nil && found {
			return box, nil
		}
		if errors.Is(err, ErrNotInstalled) {
			return TextBox{}, err
		}
		if time.Now().After(deadline) {
			return TextBox{}, fmt.Errorf("WaitForText: timeout after %dms waiting for %q", timeoutMs, needle)
		}
		time.Sleep(400 * time.Millisecond)
	}
}

// scanScreenHOCR captures the screen and runs tesseract in HOCR mode, then
// parses word boxes. Coordinates are returned in image pixels which equal
// screen pixels (Capture takes a full-screen scrot at native resolution).
func scanScreenHOCR() ([]TextBox, error) {
	if _, lookErr := exec.LookPath("tesseract"); lookErr != nil {
		return nil, ErrNotInstalled
	}

	data, err := screensvc.Capture()
	if err != nil {
		return nil, fmt.Errorf("ocr: screenshot: %w", err)
	}

	tmp, err := os.CreateTemp("", "nullhand-hocr-*.png")
	if err != nil {
		return nil, fmt.Errorf("ocr: create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return nil, fmt.Errorf("ocr: write temp file: %w", err)
	}
	tmp.Close()

	// tesseract <image> stdout -l eng hocr
	cmd := exec.Command("tesseract", tmpPath, "stdout", "-l", "eng", "hocr")
	out, err := cmd.Output()
	if err != nil {
		// tesseract may exit non-zero but still produce output; only error if empty.
		if len(out) == 0 {
			return nil, fmt.Errorf("ocr: tesseract hocr failed: %w", err)
		}
	}

	return parseHOCR(string(out)), nil
}

// parseHOCR walks the HOCR XML and returns word-level boxes. Falls back to
// line-level boxes if no words were extracted (rare).
func parseHOCR(hocr string) []TextBox {
	var boxes []TextBox
	for _, m := range hocrBboxRe.FindAllStringSubmatch(hocr, -1) {
		text := strings.TrimSpace(m[5])
		if text == "" {
			continue
		}
		x1, _ := strconv.Atoi(m[1])
		y1, _ := strconv.Atoi(m[2])
		x2, _ := strconv.Atoi(m[3])
		y2, _ := strconv.Atoi(m[4])
		if x2 <= x1 || y2 <= y1 {
			continue
		}
		boxes = append(boxes, TextBox{
			Text: text, X: x1, Y: y1, W: x2 - x1, H: y2 - y1,
			CenterX: (x1 + x2) / 2, CenterY: (y1 + y2) / 2,
		})
	}
	if len(boxes) > 0 {
		return boxes
	}

	// Fallback: line-level.
	for _, m := range hocrLineRe.FindAllStringSubmatch(hocr, -1) {
		x1, _ := strconv.Atoi(m[1])
		y1, _ := strconv.Atoi(m[2])
		x2, _ := strconv.Atoi(m[3])
		y2, _ := strconv.Atoi(m[4])
		text := strings.TrimSpace(hocrTagRe.ReplaceAllString(m[5], " "))
		if text == "" || x2 <= x1 || y2 <= y1 {
			continue
		}
		boxes = append(boxes, TextBox{
			Text: text, X: x1, Y: y1, W: x2 - x1, H: y2 - y1,
			CenterX: (x1 + x2) / 2, CenterY: (y1 + y2) / 2,
		})
	}
	return boxes
}

// mergeBoxes returns a TextBox spanning the bounding box union of the input.
func mergeBoxes(in []TextBox) TextBox {
	if len(in) == 0 {
		return TextBox{}
	}
	x1, y1 := in[0].X, in[0].Y
	x2, y2 := in[0].X+in[0].W, in[0].Y+in[0].H
	parts := make([]string, 0, len(in))
	for _, b := range in {
		if b.X < x1 {
			x1 = b.X
		}
		if b.Y < y1 {
			y1 = b.Y
		}
		if b.X+b.W > x2 {
			x2 = b.X + b.W
		}
		if b.Y+b.H > y2 {
			y2 = b.Y + b.H
		}
		parts = append(parts, b.Text)
	}
	return TextBox{
		Text: strings.Join(parts, " "),
		X:    x1, Y: y1, W: x2 - x1, H: y2 - y1,
		CenterX: (x1 + x2) / 2,
		CenterY: (y1 + y2) / 2,
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
