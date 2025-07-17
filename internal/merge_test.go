package internal

import (
	"bytes"
	"golang.org/x/tools/cover"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMergeProfilesFromTestdata(t *testing.T) {
	files := []string{"unit_coverage.txt", "integration_coverage.txt"}
	var profiles []*cover.Profile
	for _, fname := range files {
		path := filepath.Join("testdata", fname)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read %s: %v", fname, err)
		}
		parsed, err := cover.ParseProfilesFromReader(strings.NewReader(string(data)))
		if err != nil {
			t.Fatalf("failed to parse %s: %v", fname, err)
		}
		for _, p := range parsed {
			profiles = AddProfile(profiles, p)
		}
	}
	var buf bytes.Buffer
	DumpProfiles(profiles, &buf)
	output := buf.String()
	if !strings.HasPrefix(output, "mode: ") {
		t.Errorf("output missing mode line: %q", output)
	}
	if len(strings.Split(output, "\n")) < 2 {
		t.Errorf("output too short: %q", output)
	}
}
