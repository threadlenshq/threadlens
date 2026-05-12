package configfile

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

type UpdateResult struct {
	UpdatedKeys []string
}

func UpdateFile(path string, values map[string]string, managedOrder []string) (UpdateResult, error) {
	if err := validateValues(values); err != nil {
		return UpdateResult{}, err
	}
	original, err := os.ReadFile(path)
	if err != nil {
		return UpdateResult{}, err
	}

	order := managedOrder
	if len(order) == 0 {
		for key := range values {
			order = append(order, key)
		}
		sort.Strings(order)
	}

	const managedMarker = "# Scout onboarding managed values"

	seen := map[string]bool{}
	changed := map[string]bool{}
	hasManagedMarker := false
	var out []string
	scanner := bufio.NewScanner(strings.NewReader(string(original)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == managedMarker {
			hasManagedMarker = true
		}
		key, ok := parseKey(line)
		if ok {
			if value, managed := values[key]; managed {
				out = append(out, key+"="+formatValue(value))
				seen[key] = true
				changed[key] = true
				continue
			}
		}
		out = append(out, line)
	}
	if err := scanner.Err(); err != nil {
		return UpdateResult{}, err
	}

	missing := make([]string, 0)
	for _, key := range order {
		if _, ok := values[key]; ok && !seen[key] {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		if len(out) > 0 && strings.TrimSpace(out[len(out)-1]) != "" {
			out = append(out, "")
		}
		if !hasManagedMarker {
			out = append(out, managedMarker)
		}
		for _, key := range missing {
			out = append(out, key+"="+formatValue(values[key]))
			changed[key] = true
		}
	}

	contents := strings.Join(out, "\n")
	if !strings.HasSuffix(contents, "\n") {
		contents += "\n"
	}
	if err := atomicWrite(path, []byte(contents)); err != nil {
		return UpdateResult{}, err
	}

	keys := make([]string, 0, len(changed))
	for key := range changed {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return UpdateResult{UpdatedKeys: keys}, nil
}

func ReadValues(path string, keys []string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	wanted := map[string]bool{}
	for _, key := range keys {
		wanted[key] = true
	}
	values := map[string]string{}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		key, ok := parseKey(line)
		if !ok || !wanted[key] {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			values[key] = unquoteValue(strings.TrimSpace(parts[1]))
		}
	}
	return values, scanner.Err()
}

func MaskValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= 4 {
		return "••••"
	}
	return "••••" + string(runes[len(runes)-4:])
}

func parseKey(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "export ") {
		return "", false
	}
	idx := strings.Index(trimmed, "=")
	if idx <= 0 {
		return "", false
	}
	key := strings.TrimSpace(trimmed[:idx])
	if key == "" {
		return "", false
	}
	for _, r := range key {
		if !(r == '_' || r == '-' || unicode.IsDigit(r) || unicode.IsLetter(r)) {
			return "", false
		}
	}
	return key, true
}

func validateValues(values map[string]string) error {
	for key, value := range values {
		for _, r := range value {
			if r == '\n' || r == '\r' || r == 0 || (unicode.IsControl(r) && r != '\t') {
				return fmt.Errorf("%s contains unsupported control character", key)
			}
		}
	}
	return nil
}

func formatValue(value string) string {
	if value == "" {
		return ""
	}
	needsQuotes := strings.ContainsAny(value, " \t#\"'")
	if !needsQuotes {
		return value
	}
	escaped := strings.ReplaceAll(value, "\\", "\\\\")
	escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
	return "\"" + escaped + "\""
}

func unquoteValue(value string) string {
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		inner := value[1 : len(value)-1]
		inner = strings.ReplaceAll(inner, `\"`, `"`)
		inner = strings.ReplaceAll(inner, `\\`, `\`)
		return inner
	}
	if idx := strings.Index(value, " #"); idx >= 0 {
		value = strings.TrimSpace(value[:idx])
	}
	return value
}

func atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".env.tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
