package plan

import (
	"os"
	"strings"
	"testing"
)

func TestSaveAndReloadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	p := &Plan{Task: "Add OAuth login!", Text: "Step 1.\nStep 2.", Model: "opus"}

	if err := s.Save(p); err != nil {
		t.Fatalf("save: %v", err)
	}
	if !strings.HasSuffix(p.Path, "-add-oauth-login.md") {
		t.Fatalf("unexpected path/slug: %s", p.Path)
	}
	raw, _ := os.ReadFile(p.Path)
	if !strings.Contains(string(raw), "# xocode plan") || !strings.Contains(string(raw), "Add OAuth login") {
		t.Fatalf("header missing from doc:\n%s", raw)
	}

	// Simulate a user edit to the body, then reload.
	edited := string(raw) + "\nStep 3 added by user.\n"
	os.WriteFile(p.Path, []byte(edited), 0o644)
	if err := s.Reload(p); err != nil {
		t.Fatalf("reload: %v", err)
	}
	if !strings.Contains(p.Text, "Step 3 added by user") || strings.Contains(p.Text, "# xocode plan") {
		t.Fatalf("reload did not strip header / keep body: %q", p.Text)
	}
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Add OAuth login!":       "add-oauth-login",
		"   ":                    "plan",
		"UPPER_case With  Space": "upper-case-with-space",
	}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
}
