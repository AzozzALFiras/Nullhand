package local

import "testing"

func TestParseAuthorRequest(t *testing.T) {
	cases := []struct {
		name      string
		input     string
		wantName  string
		wantSteps int
		wantErr   bool
	}{
		{
			name:      "english save recipe",
			input:     "save this as recipe morning_routine: open Firefox, type hello, send",
			wantName:  "morning_routine",
			wantSteps: 3,
		},
		{
			name:      "english with spaces in name normalises",
			input:     "save recipe Morning Routine: open Firefox, send",
			wantName:  "morning_routine",
			wantSteps: 2,
		},
		{
			name:      "arabic save routine",
			input:     "احفظ هذا كروتين الصباح: افتح Firefox، اكتب hello، ارسل",
			wantName:  "الصباح",
			wantSteps: 3,
		},
		{
			name:      "arabic alt phrasing",
			input:     "احفظ روتين العمل: افتح Firefox، افتح Slack",
			wantName:  "العمل",
			wantSteps: 2,
		},
		{
			name:    "unsupported step (run_recipe nesting)",
			input:   "save recipe nested: recipe whatsapp_send_message",
			wantErr: true,
		},
		{
			name:  "not a save request",
			input: "open Firefox",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := ParseAuthorRequest(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got req=%+v", req)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantName == "" {
				if req != nil {
					t.Fatalf("expected nil req for non-author input, got %+v", req)
				}
				return
			}
			if req == nil {
				t.Fatalf("expected req with name=%q, got nil", tc.wantName)
			}
			if req.Name != tc.wantName {
				t.Errorf("name = %q, want %q", req.Name, tc.wantName)
			}
			if len(req.Recipe.Steps) != tc.wantSteps {
				t.Errorf("step count = %d, want %d (steps=%+v)", len(req.Recipe.Steps), tc.wantSteps, req.Recipe.Steps)
			}
		})
	}
}
