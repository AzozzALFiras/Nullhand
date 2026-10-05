package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// pngBytes is a 1x1 PNG: something every platform previews as an image.
var pngBytes = []byte{
	0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a,
	0x00, 0x00, 0x00, 0x0d, 'I', 'H', 'D', 'R',
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
	0x89, 0x00, 0x00, 0x00, 0x0a, 'I', 'D', 'A', 'T',
	0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00, 0x05,
	0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00, 0x00,
	0x00, 0x00, 'I', 'E', 'N', 'D', 0xae, 0x42, 0x60, 0x82,
}

func writePNG(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "dot.png")
	if err := os.WriteFile(path, pngBytes, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestFindFilesToolRejectsEmptyQuery(t *testing.T) {
	vm := &ViewModel{}
	parts, err, handled := vm.executeSearchTool(macCall("find_files", map[string]string{"folder": "~"}), nil)
	if !handled || err == nil {
		t.Fatalf("an empty query must be refused: handled=%v err=%v", handled, err)
	}
	if text := partsText(parts); !strings.HasPrefix(text, "⚠️") {
		t.Errorf("the offline parser only surfaces ⚠️/❌/ℹ️ results, got %q", text)
	}
}

func TestFindFilesToolListsHits(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	vm := &ViewModel{}
	parts, err, handled := vm.executeSearchTool(macCall("find_files", map[string]string{
		"query":  "search_tools.go",
		"folder": wd,
	}), nil)
	if !handled || err != nil {
		t.Fatalf("handled=%v err=%v", handled, err)
	}
	text := partsText(parts)
	if !strings.Contains(text, "search_tools.go") {
		t.Errorf("expected the matching path in the result, got %q", text)
	}
	if !strings.HasPrefix(text, "ℹ️") {
		t.Errorf("search results are for the user to read, got %q", text)
	}
}

func TestPreviewFileToolDeliversImage(t *testing.T) {
	var delivered []byte
	var caption string
	sendPhoto := func(data []byte, c string) error {
		delivered, caption = data, c
		return nil
	}

	path := writePNG(t)
	vm := &ViewModel{}
	parts, err, handled := vm.executeSearchTool(macCall("preview_file", map[string]string{"path": path}), sendPhoto)
	if !handled || err != nil {
		t.Fatalf("handled=%v err=%v", handled, err)
	}
	if len(delivered) == 0 {
		t.Error("the image should have been sent to the chat")
	}
	if !strings.Contains(caption, path) {
		t.Errorf("caption should name the file, got %q", caption)
	}
	if text := partsText(parts); !strings.Contains(text, "cannot see") {
		t.Errorf("the model should be told it cannot see the image, got %q", text)
	}
}

func TestPreviewFileToolNeedsPhotoDelivery(t *testing.T) {
	vm := &ViewModel{}
	_, err, handled := vm.executeSearchTool(macCall("preview_file", map[string]string{"path": writePNG(t)}), nil)
	if !handled || err == nil {
		t.Errorf("without a photo callback an image preview cannot be delivered: handled=%v err=%v", handled, err)
	}
}

func TestPreviewFileToolReportsMissingFile(t *testing.T) {
	vm := &ViewModel{}
	_, err, handled := vm.executeSearchTool(macCall("preview_file", map[string]string{
		"path": filepath.Join(t.TempDir(), "nope.pdf"),
	}), nil)
	if !handled || err == nil {
		t.Errorf("a missing file must be reported: handled=%v err=%v", handled, err)
	}
}

func TestExecuteSearchToolIgnoresOtherTools(t *testing.T) {
	vm := &ViewModel{}
	if _, _, handled := vm.executeSearchTool(macCall("run_shell", map[string]string{"command": "ls"}), nil); handled {
		t.Error("unrelated tools must fall through to the main handler")
	}
}

func TestSearchToolDefinitions(t *testing.T) {
	defs := searchToolDefinitions()
	seen := map[string]bool{}
	for _, d := range defs {
		if d.Name == "" || d.Description == "" {
			t.Errorf("tool %+v needs a name and a description", d)
		}
		for _, p := range d.Parameters {
			if p.Name == "" || p.Description == "" {
				t.Errorf("tool %q has an undocumented parameter %+v", d.Name, p)
			}
		}
		seen[d.Name] = true
	}
	for _, required := range []string{"find_files", "preview_file"} {
		if !seen[required] {
			t.Errorf("missing tool definition %q", required)
		}
	}
}

// These tools must be offered everywhere, not only on macOS.
func TestSearchToolsAreInTheDefaultToolList(t *testing.T) {
	var names []string
	for _, d := range New(nil, nil).tools {
		names = append(names, d.Name)
	}
	joined := strings.Join(names, " ")
	for _, want := range []string{"find_files", "preview_file"} {
		if !strings.Contains(joined, want) {
			t.Errorf("tool list is missing %q: %v", want, names)
		}
	}
}
