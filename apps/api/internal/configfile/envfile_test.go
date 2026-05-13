package configfile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateEnvFilePreservesCommentsUpdatesAndAppends(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	initial := "# Scout env\nANTHROPIC_API_KEY=old\n\n# Keep me\nPORT=4749\n"
	if err := os.WriteFile(path, []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := UpdateFile(path, map[string]string{
		"ANTHROPIC_API_KEY": "sk-ant-new",
		"PARALLEL_API_KEY":  "parallel-123",
	}, []string{"ANTHROPIC_API_KEY", "PARALLEL_API_KEY"})
	if err != nil {
		t.Fatalf("UpdateFile: %v", err)
	}
	if len(result.UpdatedKeys) != 2 {
		t.Fatalf("UpdatedKeys = %#v, want 2 keys", result.UpdatedKeys)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{
		"# Scout env",
		"ANTHROPIC_API_KEY=sk-ant-new",
		"# Keep me",
		"PORT=4749",
		"# Scout onboarding managed values",
		"PARALLEL_API_KEY=parallel-123",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("updated env missing %q:\n%s", want, text)
		}
	}
}

func TestUpdateEnvFileRejectsDotenvInjection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("ANTHROPIC_API_KEY=\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := UpdateFile(path, map[string]string{"ANTHROPIC_API_KEY": "abc\nMALICIOUS=1"}, []string{"ANTHROPIC_API_KEY"})
	if err == nil || !strings.Contains(err.Error(), "unsupported control character") {
		t.Fatalf("error = %v, want unsupported control character", err)
	}
}

func TestUpdateEnvFileQuotesWhitespaceAndHashValues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := UpdateFile(path, map[string]string{"SCOUT_AI_BRIDGE_URL": "http://localhost:4761/path#token"}, []string{"SCOUT_AI_BRIDGE_URL"})
	if err != nil {
		t.Fatalf("UpdateFile: %v", err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), `SCOUT_AI_BRIDGE_URL="http://localhost:4761/path#token"`) {
		t.Fatalf("expected quoted bridge URL, got:\n%s", string(data))
	}
}

func TestMaskValueNeverReturnsRawSecret(t *testing.T) {
	masked := MaskValue("sk-ant-1234567890abcdef")
	if masked == "" || strings.Contains(masked, "1234567890") || strings.Contains(masked, "sk-ant-1234567890abcdef") {
		t.Fatalf("masked value leaked secret: %q", masked)
	}
	if !strings.HasSuffix(masked, "cdef") {
		t.Fatalf("masked value = %q, want last four chars visible", masked)
	}
}
