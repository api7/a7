//go:build e2e

package skills

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
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

	a7Binary = filepath.Join(tmpDir, "a7")
	cmd := exec.Command("go", "build", "-o", a7Binary, "./cmd/a7")
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		_ = os.RemoveAll(tmpDir)
		os.Exit(1)
	}

	exitCode := m.Run()
	if err := os.RemoveAll(tmpDir); err != nil && exitCode == 0 {
		fmt.Fprintf(os.Stderr, "failed to remove temp dir %s: %v\n", tmpDir, err)
		exitCode = 1
	}
	os.Exit(exitCode)
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
	Fields             map[string]string
	A7Commands         []string
	HasDescriptionText bool
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
	frontmatterLines := lines[1:end]
	for i, line := range frontmatterLines {
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
		if key == "description" {
			metadata.HasDescriptionText = hasNonEmptyDescription(frontmatterLines, i, value)
		}
		if key == "a7_commands" {
			inA7Commands = true
		}
	}
	return metadata
}

func hasNonEmptyDescription(lines []string, startIdx int, value string) bool {
	value = strings.Trim(strings.TrimSpace(value), `"`)
	if value != "" && !strings.HasPrefix(value, ">") && !strings.HasPrefix(value, "|") {
		return true
	}
	for _, line := range lines[startIdx+1:] {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if !strings.HasPrefix(line, " ") && strings.Contains(trimmed, ":") {
			return false
		}
		if trimmed != ">" && trimmed != ">-" && trimmed != "|" && trimmed != "|-" {
			return true
		}
	}
	return false
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
		if !metadata.HasDescriptionText {
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
			command = strings.TrimSpace(command)
			if command == "" {
				continue
			}
			if command != "a7" && !strings.HasPrefix(command, "a7 ") {
				t.Fatalf("%s: a7_commands entry %q must start with a7", file, command)
			}
			helpCommand := strconv.Quote(a7Binary) + strings.TrimPrefix(command, "a7") + " --help"
			cmd := exec.Command("sh", "-c", helpCommand)
			cmd.Dir = root
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("%s: command %q is not supported by current a7 CLI: %v\n%s", file, command, err, string(output))
			}
		}
	}
}

func TestSkillShellExamplesUseSupportedA7CommandsAndFlags(t *testing.T) {
	shellFencePattern := regexp.MustCompile("(?s)```(?:bash|sh|shell)\\s*\\n(.*?)```")
	longFlagPattern := regexp.MustCompile(`--[a-z][a-z0-9-]*`)
	invocationPattern := regexp.MustCompile(`(?:^|[^A-Za-z0-9_-])(a7(?:\s+[^|;&)]*)?)`)
	valueFlags := a7GlobalValueFlags()
	root := repoRoot(t)
	matches, err := filepath.Glob(filepath.Join(root, "skills", "*", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) == 0 {
		t.Fatal("expected at least one skill file")
	}
	rootHelp := commandHelp(t, nil)
	rootCommands := availableCommands(rootHelp)
	rootFlags := availableFlags(rootHelp, longFlagPattern)
	regressionPath, regressionHelp, regressionErr := resolveCommand(t, "regression", []string{"route", "--gateway-group", "default", "craete"}, rootCommands, rootFlags, valueFlags)
	if regressionErr == nil {
		t.Fatalf("expected misspelled command after a persistent flag to fail, got path %q and help %q", regressionPath, regressionHelp)
	}
	embedded := cliInvocations("CURRENT=$(a7 route get example -g default)", invocationPattern)
	if len(embedded) != 1 || !strings.HasPrefix(embedded[0], "a7 route get") {
		t.Fatalf("expected embedded a7 invocation, got %q", embedded)
	}
	for _, file := range matches {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, block := range shellFencePattern.FindAllStringSubmatch(string(data), -1) {
			for _, line := range joinedShellLines(block[1]) {
				for _, invocation := range cliInvocations(line, invocationPattern) {
					fields := strings.Fields(invocation)
					if len(fields) < 2 {
						continue
					}
					path, help, err := resolveCommand(t, file, commandFields(fields[1:]), rootCommands, rootFlags, valueFlags)
					if err != nil {
						t.Fatalf("%s: %v", file, err)
					}
					validFlags := mergeFlagSets(rootFlags, availableFlags(help, longFlagPattern))
					for _, flag := range longFlagPattern.FindAllString(invocation, -1) {
						if flag == "--help" {
							continue
						}
						if !validFlags[flag] {
							t.Fatalf("%s: command %q uses unsupported flag %q", file, "a7 "+strings.Join(path, " "), flag)
						}
					}
				}
			}
		}
	}
}

func commandFields(fields []string) []string {
	valueFlags := a7GlobalValueFlags()
	for len(fields) > 0 && strings.HasPrefix(fields[0], "-") {
		flag := strings.SplitN(fields[0], "=", 2)[0]
		hasInlineValue := strings.Contains(fields[0], "=")
		fields = fields[1:]
		if valueFlags[flag] && !hasInlineValue && len(fields) > 0 {
			fields = fields[1:]
		}
	}
	return fields
}

func a7GlobalValueFlags() map[string]bool {
	return map[string]bool{
		"--gateway-group": true,
		"--output":        true,
		"--server":        true,
		"--token":         true,
		"-g":              true,
		"-o":              true,
	}
}

func commandHelp(t *testing.T, path []string) string {
	t.Helper()
	args := append(append([]string{}, path...), "--help")
	output, err := exec.Command(a7Binary, args...).CombinedOutput()
	if err != nil {
		t.Fatalf("a7 %s --help failed: %v\n%s", strings.Join(path, " "), err, output)
	}
	return string(output)
}

func availableCommands(help string) map[string]bool {
	commands := map[string]bool{}
	inCommands := false
	for _, line := range strings.Split(help, "\n") {
		if strings.TrimSpace(line) == "Available Commands:" {
			inCommands = true
			continue
		}
		if !inCommands {
			continue
		}
		if strings.TrimSpace(line) == "" {
			break
		}
		fields := strings.Fields(line)
		if len(fields) > 0 {
			commands[fields[0]] = true
		}
	}
	return commands
}

func availableFlags(help string, longFlagPattern *regexp.Regexp) map[string]bool {
	flags := map[string]bool{}
	inFlags := false
	for _, line := range strings.Split(help, "\n") {
		heading := strings.TrimSpace(line)
		if heading == "Flags:" || heading == "Global Flags:" {
			inFlags = true
			continue
		}
		if heading == "" {
			inFlags = false
			continue
		}
		if !inFlags {
			continue
		}
		if flag := longFlagPattern.FindString(line); flag != "" {
			flags[flag] = true
		}
	}
	return flags
}

func mergeFlagSets(first map[string]bool, second map[string]bool) map[string]bool {
	merged := map[string]bool{}
	for flag := range first {
		merged[flag] = true
	}
	for flag := range second {
		merged[flag] = true
	}
	return merged
}

func commandFieldIsSubcommand(subcommands map[string]bool, field string) (bool, error) {
	if len(subcommands) == 0 {
		return false, nil
	}
	if !subcommands[field] {
		return false, fmt.Errorf("unsupported nested command %q", field)
	}
	return true, nil
}

func TestCommandFieldIsSubcommand_RejectsUnknownNestedCommand(t *testing.T) {
	subcommands := map[string]bool{"create": true, "list": true}
	if _, err := commandFieldIsSubcommand(subcommands, "crte"); err == nil {
		t.Fatal("expected misspelled nested command to fail")
	}
}

func resolveCommand(t *testing.T, file string, fields []string, commands map[string]bool, rootFlags map[string]bool, valueFlags map[string]bool) ([]string, string, error) {
	t.Helper()
	if len(fields) == 0 || !commands[fields[0]] {
		return nil, "", fmt.Errorf("unsupported a7 command %q", strings.Join(fields, " "))
	}
	path := []string{fields[0]}
	help := commandHelp(t, path)
	for index := 1; index < len(fields); {
		field := fields[index]
		if strings.ContainsAny(field, "|<>") {
			break
		}
		subcommands := availableCommands(help)
		if len(subcommands) == 0 {
			break
		}
		if strings.HasPrefix(field, "-") {
			flag := strings.SplitN(field, "=", 2)[0]
			if !rootFlags[flag] {
				return path, help, fmt.Errorf("unsupported interspersed flag %q before a7 subcommand", flag)
			}
			index++
			if valueFlags[flag] && !strings.Contains(field, "=") {
				if index >= len(fields) {
					return path, help, fmt.Errorf("flag %q requires a value", flag)
				}
				index++
			}
			continue
		}
		isSubcommand, err := commandFieldIsSubcommand(subcommands, field)
		if err != nil {
			return path, help, fmt.Errorf("a7 %s: %w", strings.Join(path, " "), err)
		}
		if !isSubcommand {
			break
		}
		path = append(path, field)
		help = commandHelp(t, path)
		index++
	}
	return path, help, nil
}

func cliInvocations(line string, invocationPattern *regexp.Regexp) []string {
	var invocations []string
	for _, match := range invocationPattern.FindAllStringSubmatch(line, -1) {
		if len(match) > 1 {
			invocations = append(invocations, strings.TrimSpace(match[1]))
		}
	}
	return invocations
}

func joinedShellLines(block string) []string {
	var commands []string
	var current string
	for _, raw := range strings.Split(block, "\n") {
		line := strings.TrimSpace(raw)
		if current == "" && (line == "" || strings.HasPrefix(line, "#")) {
			continue
		}
		current += " " + strings.TrimSuffix(line, "\\")
		if strings.HasSuffix(line, "\\") {
			continue
		}
		commands = append(commands, strings.TrimSpace(current))
		current = ""
	}
	if strings.TrimSpace(current) != "" {
		commands = append(commands, strings.TrimSpace(current))
	}
	return commands
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
		"a7 upstream",
		"a7 upstream health",
		"a7 consumer-group",
		"a7 service-template",
		"a7 consumer-restriction create",
		"\nupstreams:",
		"\n    upstreams:",
		"upstream_id:",
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
	entries, err := os.ReadDir(filepath.Join(root, "skills"))
	if err != nil {
		t.Fatal(err)
	}
	existing := map[string]bool{}
	for _, entry := range entries {
		if entry.IsDir() {
			existing[entry.Name()] = true
		}
	}

	data, err := os.ReadFile(filepath.Join(root, "docs", "skills.md"))
	if err != nil {
		t.Fatal(err)
	}
	doc := string(data)
	referencedSkills := regexp.MustCompile(`\ba7-[a-z0-9]+(?:-[a-z0-9]+)*\b`).FindAllString(doc, -1)
	categoryNames := map[string]bool{
		"a7-persona": true,
		"a7-plugin":  true,
		"a7-recipe":  true,
	}
	missing := map[string]bool{}
	for _, name := range referencedSkills {
		if categoryNames[name] {
			continue
		}
		if !existing[name] {
			missing[name] = true
		}
	}
	if len(missing) == 0 {
		return
	}
	names := make([]string, 0, len(missing))
	for name := range missing {
		names = append(names, name)
	}
	sort.Strings(names)
	t.Fatalf("docs/skills.md references missing skills: %s", strings.Join(names, ", "))
}
