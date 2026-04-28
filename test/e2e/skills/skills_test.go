//go:build e2e

package skills

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var skillNamePattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("failed to locate repository root")
		}
		dir = parent
	}
}

func frontmatter(t *testing.T, file string) map[string]string {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) < 3 || lines[0] != "---" {
		t.Fatalf("%s: missing opening frontmatter delimiter", file)
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			end = i
			break
		}
	}
	if end == -1 {
		t.Fatalf("%s: missing closing frontmatter delimiter", file)
	}
	fields := map[string]string{}
	for _, line := range lines[1:end] {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"`)
		if key != "" && value != "" {
			fields[key] = value
		}
	}
	return fields
}

func TestSkillFrontmatterMatchesDirectories(t *testing.T) {
	root := repoRoot(t)
	entries, err := os.ReadDir(filepath.Join(root, "skills"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least one skill")
	}
	seen := map[string]bool{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		file := filepath.Join(root, "skills", name, "SKILL.md")
		fields := frontmatter(t, file)
		if fields["name"] != name {
			t.Fatalf("%s: frontmatter name %q must match directory name", file, fields["name"])
		}
		if !skillNamePattern.MatchString(fields["name"]) {
			t.Fatalf("%s: skill name must be kebab-case", file)
		}
		if fields["description"] == "" {
			t.Fatalf("%s: description is required", file)
		}
		if seen[fields["name"]] {
			t.Fatalf("duplicate skill name %q", fields["name"])
		}
		seen[fields["name"]] = true
	}
}

func TestSkillsDoNotReferenceRemovedA7Commands(t *testing.T) {
	root := repoRoot(t)
	disallowed := []string{
		"a7 health",
		"a7 portal",
		"a7 upstream health",
		"a7 consumer-restriction create",
	}
	for _, pattern := range disallowed {
		matches, err := filepath.Glob(filepath.Join(root, "skills", "*", "SKILL.md"))
		if err != nil {
			t.Fatal(err)
		}
		for _, file := range matches {
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(data), pattern) {
				t.Fatalf("%s: references removed or unsupported command %q", file, pattern)
			}
		}
	}
}

func TestSkillsDocumentationReferencesExistingSkills(t *testing.T) {
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "docs", "skills.md"))
	if err != nil {
		t.Fatal(err)
	}
	doc := string(data)
	staleSkillNames := []string{
		"a7-recipe-gateway-group",
		"a7-persona-platform-eng",
		"a7-recipe-service-template",
		"a7-plugin-ai-rag",
		"a7-plugin-ai-token-limiter",
		"a7-recipe-service-registry",
	}
	for _, name := range staleSkillNames {
		if strings.Contains(doc, name) {
			t.Fatalf("docs/skills.md references missing skill %q", name)
		}
	}
}
