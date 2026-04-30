package local

import "testing"

// TestParserArabicFlows verifies the Arabic-language flows introduced for
// WhatsApp send, Settings search, button click, browser navigation. These
// were the four user-facing failures reported.
func TestParserArabicFlows(t *testing.T) {
	cases := []struct {
		input         string
		wantTool      string
		wantRecipe    string // when wantTool == "run_recipe"
		wantArg       string // key whose value to verify
		wantArgValue  string // expected value (substring)
	}{
		// ── WhatsApp messaging ────────────────────────────────────
		{
			input:        "افتح واتساب وأرسل لعزوز رسالة مرحبا",
			wantTool:     "run_recipe",
			wantRecipe:   "whatsapp_send_message",
			wantArg:      "params_json",
			wantArgValue: `"contact":"عزوز"`,
		},
		{
			input:        "ارسل لعزوز في الواتساب: مرحبا",
			wantTool:     "run_recipe",
			wantRecipe:   "whatsapp_send_message",
			wantArg:      "params_json",
			wantArgValue: `"message":"مرحبا"`,
		},
		{
			input:        "واتساب عزوز: مرحبا",
			wantTool:     "run_recipe",
			wantRecipe:   "whatsapp_send_message",
			wantArg:      "params_json",
			wantArgValue: `"contact":"عزوز"`,
		},

		// ── Settings search and panel ────────────────────────────
		{
			input:        "ابحث في الإعدادات عن WiFi",
			wantTool:     "run_recipe",
			wantRecipe:   "settings_search",
			wantArg:      "params_json",
			wantArgValue: `"query":"WiFi"`,
		},
		{
			input:        "search settings for bluetooth",
			wantTool:     "run_recipe",
			wantRecipe:   "settings_search",
			wantArg:      "params_json",
			wantArgValue: `"query":"bluetooth"`,
		},

		// ── Click button (Arabic + English) ──────────────────────
		{
			input:        "اضغط زر إرسال",
			wantTool:     "run_recipe",
			wantRecipe:   "click_button",
			wantArg:      "params_json",
			wantArgValue: `"label":"إرسال"`,
		},
		{
			input:        "click the Send button",
			wantTool:     "run_recipe",
			wantRecipe:   "click_button",
			wantArg:      "params_json",
			wantArgValue: `"label":"Send"`,
		},

		// ── Browser navigation (Arabic + English) ────────────────
		{
			input:        "افتح فايرفوكس وروح إلى github.com",
			wantTool:     "run_recipe",
			wantRecipe:   "browser_open_url",
			wantArg:      "params_json",
			wantArgValue: `"url":"github.com"`,
		},
		{
			input:        "open firefox and go to github.com",
			wantTool:     "run_recipe",
			wantRecipe:   "browser_open_url",
			wantArg:      "params_json",
			wantArgValue: `"url":"github.com"`,
		},
		{
			input:      "ارجع",
			wantTool:   "run_recipe",
			wantRecipe: "browser_back",
		},
		{
			input:      "علامة تبويب جديدة",
			wantTool:   "run_recipe",
			wantRecipe: "browser_new_tab",
		},
		{
			input:      "تحديث",
			wantTool:   "run_recipe",
			wantRecipe: "browser_reload",
		},
	}

	for _, tc := range cases {
		got := Parse(tc.input)
		if len(got) == 0 {
			t.Errorf("Parse(%q) returned no tool calls", tc.input)
			continue
		}
		if got[0].ToolName != tc.wantTool {
			t.Errorf("Parse(%q) tool = %s, want %s", tc.input, got[0].ToolName, tc.wantTool)
			continue
		}
		if tc.wantTool == "run_recipe" && tc.wantRecipe != "" {
			if got[0].Arguments["name"] != tc.wantRecipe {
				t.Errorf("Parse(%q) recipe = %s, want %s", tc.input, got[0].Arguments["name"], tc.wantRecipe)
			}
		}
		if tc.wantArg != "" {
			val := got[0].Arguments[tc.wantArg]
			if !contains(val, tc.wantArgValue) {
				t.Errorf("Parse(%q) arg[%s] = %q, want substring %q", tc.input, tc.wantArg, val, tc.wantArgValue)
			}
		}
	}
}

// contains is a thin wrapper avoiding an extra import in tests.
func contains(haystack, needle string) bool {
	if needle == "" {
		return true
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
