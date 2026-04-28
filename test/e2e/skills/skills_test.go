//go:build e2e

package skills

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var skillNamePattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
var a7Binary string

func locateRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

func TestMain(m *testing.M) {
	root, err := locateRepoRoot()
	if err != nil {
		os.Exit(1)
	}
	tmpDir, err := os.MkdirTemp("", "a7-skills-test-*")
	if err != nil {
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	a7Binary = filepath.Join(tmpDir, "a7")
	cmd := exec.Command("go", "build", "-o", a7Binary, "./cmd/a7")
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := locateRepoRoot()
	if err != nil {
		t.Fatal("failed to locate repository root")
	}
	return root
}

type skillMetadata struct {
	Fields     map[string]string
	A7Commands []string
}

func frontmatter(t *testing.T, file string) skillMetadata {
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
	metadata := skillMetadata{Fields: map[string]string{}}
	inA7Commands := false
	for _, line := range lines[1:end] {
		key, value, ok := strings.Cut(line, ":")
		trimmed := strings.TrimSpace(line)
		if inA7Commands {
			if strings.HasPrefix(trimmed, "- ") {
				command := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
				if command != "" {
					metadata.A7Commands = append(metadata.A7Commands, command)
				}
				continue
			}
			if trimmed != "" && !strings.HasPrefix(line, "    ") {
				inA7Commands = false
			}
		}
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"`)
		if key != "" && value != "" {
			metadata.Fields[key] = value
		}
		if key == "a7_commands" {
			inA7Commands = true
		}
	}
	return metadata
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
		metadata := frontmatter(t, file)
		fields := metadata.Fields
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

func TestSkillDeclaredA7CommandsExist(t *testing.T) {
	root := repoRoot(t)
	matches, err := filepath.Glob(filepath.Join(root, "skills", "*", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range matches {
		metadata := frontmatter(t, file)
		for _, command := range metadata.A7Commands {
			fields := strings.Fields(command)
			if len(fields) == 0 {
				continue
			}
			if fields[0] != "a7" {
				t.Fatalf("%s: a7_commands entry %q must start with a7", file, command)
			}
			args := append(fields[1:], "--help")
			cmd := exec.Command(a7Binary, args...)
			cmd.Dir = root
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("%s: command %q is not supported by current a7 CLI: %v\n%s", file, command, err, string(output))
			}
		}
	}
}

func TestPluginSkillsDeclarePluginName(t *testing.T) {
	root := repoRoot(t)
	matches, err := filepath.Glob(filepath.Join(root, "skills", "a7-plugin-*", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range matches {
		metadata := frontmatter(t, file)
		if metadata.Fields["plugin_name"] == "" {
			t.Fatalf("%s: plugin skills must declare metadata.plugin_name", file)
		}
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
