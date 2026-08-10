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
	"unicode"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"

	"github.com/api7/a7/internal/config"
	cmd "github.com/api7/a7/pkg/cmd"
	rootcmd "github.com/api7/a7/pkg/cmd/root"
	"github.com/api7/a7/pkg/cmdutil"
	"github.com/api7/a7/pkg/iostreams"
)

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

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := locateRepoRoot()
	if err != nil {
		t.Fatal("failed to locate repository root")
	}
	return root
}

func buildA7Binary(t *testing.T, root string) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "a7")
	cmd := exec.Command("go", "build", "-o", binary, "./cmd/a7")
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build a7: %v\n%s", err, output)
	}
	return binary
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
	skillNamePattern := regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
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

func TestSkillCommands(t *testing.T) {
	root := repoRoot(t)
	binary := buildA7Binary(t, root)
	commandTree := newA7CommandTree()

	t.Run("DeclaredA7CommandsExist", func(t *testing.T) {
		testSkillDeclaredA7CommandsExist(t, root, binary)
	})
	t.Run("ExamplesUseSupportedA7CommandsAndFlags", func(t *testing.T) {
		testSkillExamplesUseSupportedA7CommandsAndFlags(t, root, binary, commandTree)
	})
}

func testSkillDeclaredA7CommandsExist(t *testing.T, root, binary string) {
	t.Helper()
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
			helpCommand := strconv.Quote(binary) + strings.TrimPrefix(command, "a7") + " --help"
			cmd := exec.Command("sh", "-c", helpCommand)
			cmd.Dir = root
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("%s: command %q is not supported by current a7 CLI: %v\n%s", file, command, err, string(output))
			}
		}
	}
}

func testSkillExamplesUseSupportedA7CommandsAndFlags(t *testing.T, root, binary string, commandTree *cobra.Command) {
	t.Helper()
	shellFencePattern := regexp.MustCompile("(?s)```(?:bash|sh|shell)\\s*\\n(.*?)```")
	yamlFencePattern := regexp.MustCompile("(?s)```(?:yaml|yml)\\s*\\n(.*?)```")
	invocationPattern := regexp.MustCompile(`(?:^|[^A-Za-z0-9_-])(a7)(?:\s|$)`)
	workflowExpressionPattern := regexp.MustCompile(`\$\{\{.*?\}\}`)
	rootFlags, valueFlags := rootFlagSets(commandTree)
	matches, err := filepath.Glob(filepath.Join(root, "skills", "*", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) == 0 {
		t.Fatal("expected at least one skill file")
	}
	rootHelp := commandHelp(t, binary, nil)
	rootCommands := availableCommands(rootHelp)
	regressionPath, regressionHelp, _, regressionErr := resolveCommand(t, binary, "regression", []string{"route", "--gateway-group", "default", "craete"}, rootCommands, rootFlags, valueFlags)
	if regressionErr == nil {
		t.Fatalf("expected misspelled command after a persistent flag to fail, got path %q and help %q", regressionPath, regressionHelp)
	}
	_, _, remaining, err := resolveCommand(t, binary, "regression", []string{"route", "get", "one", "two"}, rootCommands, rootFlags, valueFlags)
	if err != nil {
		t.Fatal(err)
	}
	if err := validatePositionalArgs(commandTree, []string{"route", "get"}, remaining); err == nil {
		t.Fatal("expected extra positional argument to fail")
	}
	_, _, remaining, err = resolveCommand(t, binary, "regression", []string{"route", "get"}, rootCommands, rootFlags, valueFlags)
	if err != nil {
		t.Fatal(err)
	}
	if err := validatePositionalArgs(commandTree, []string{"route", "get"}, remaining); err == nil {
		t.Fatal("expected missing positional argument to fail")
	}
	_, _, remaining, err = resolveCommand(t, binary, "regression", []string{"route", "list", "--output", "wide"}, rootCommands, rootFlags, valueFlags)
	if err != nil {
		t.Fatal(err)
	}
	if err := validatePositionalArgs(commandTree, []string{"route", "list"}, remaining); err == nil {
		t.Fatal("expected unsupported output format to fail")
	}
	_, _, remaining, err = resolveCommand(t, binary, "regression", []string{"debug", "trace", "<route-id>", "--bogus"}, rootCommands, rootFlags, valueFlags)
	if err != nil {
		t.Fatal(err)
	}
	if err := validatePositionalArgs(commandTree, []string{"debug", "trace"}, remaining); err == nil {
		t.Fatal("expected unsupported flag after an angle-bracket placeholder to fail")
	}
	for _, fields := range [][]string{
		{"--bogus", "route", "list"},
		{"--bogus=value", "route", "list"},
	} {
		if _, err := commandFields(fields, commandTree, rootFlags, valueFlags); err == nil {
			t.Fatalf("expected unsupported root flag in %q to fail", fields)
		}
	}
	fields, err := commandFields([]string{"--output", "json", "route", "list"}, commandTree, rootFlags, valueFlags)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(fields, " ") != "route list" {
		t.Fatalf("expected supported root flag to be removed, got %q", fields)
	}
	embedded := cliInvocations("CURRENT=$(a7 route get example -g default)", invocationPattern)
	if len(embedded) != 1 || !strings.HasPrefix(embedded[0], "a7 route get") {
		t.Fatalf("expected embedded a7 invocation, got %q", embedded)
	}
	quoted := cliInvocations(`a7 debug trace id --header "X-Test: a|b;c&d)" --bogus`, invocationPattern)
	if len(quoted) != 1 || !strings.Contains(quoted[0], "--bogus") {
		t.Fatalf("expected quoted separators to preserve the complete invocation, got %q", quoted)
	}
	yamlBlocks, err := skillShellBlocks("```yaml\n- name: Validate\n  run: >\n    a7 route list\n    --unsupported\n```", shellFencePattern, yamlFencePattern)
	if err != nil {
		t.Fatalf("failed to extract workflow run block: %v", err)
	}
	yamlCommands := joinedShellLines(yamlBlocks[0])
	if len(yamlCommands) != 1 || yamlCommands[0] != "a7 route list --unsupported" {
		t.Fatalf("expected workflow run block, got %q", yamlBlocks)
	}
	for _, file := range matches {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		blocks, err := skillShellBlocks(string(data), shellFencePattern, yamlFencePattern)
		if err != nil {
			t.Fatalf("%s: failed to parse fenced YAML: %v", file, err)
		}
		for _, block := range blocks {
			for _, line := range joinedShellLines(block) {
				for _, invocation := range cliInvocations(line, invocationPattern) {
					fields, err := shellFields(workflowExpressionPattern.ReplaceAllString(invocation, "workflow-expression"))
					if err != nil {
						t.Fatalf("%s: cannot parse command %q: %v", file, invocation, err)
					}
					if len(fields) < 2 {
						continue
					}
					commandArgs, err := commandFields(fields[1:], commandTree, rootFlags, valueFlags)
					if err != nil {
						t.Fatalf("%s: %v", file, err)
					}
					path, _, remaining, err := resolveCommand(t, binary, file, commandArgs, rootCommands, rootFlags, valueFlags)
					if err != nil {
						t.Fatalf("%s: %v", file, err)
					}
					if err := validatePositionalArgs(commandTree, path, remaining); err != nil {
						t.Fatalf("%s: command %q uses invalid positional arguments: %v", file, "a7 "+strings.Join(path, " "), err)
					}
				}
			}
		}
	}
}

func skillShellBlocks(data string, shellFencePattern, yamlFencePattern *regexp.Regexp) ([]string, error) {
	var blocks []string
	for _, match := range shellFencePattern.FindAllStringSubmatch(data, -1) {
		blocks = append(blocks, match[1])
	}
	for _, match := range yamlFencePattern.FindAllStringSubmatch(data, -1) {
		runBlocks, err := yamlRunBlocks(match[1])
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, runBlocks...)
	}
	return blocks, nil
}

func yamlRunBlocks(block string) ([]string, error) {
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(block), &root); err != nil {
		return nil, err
	}
	var runBlocks []string
	collectYAMLRunBlocks(&root, &runBlocks)
	return runBlocks, nil
}

func collectYAMLRunBlocks(node *yaml.Node, runBlocks *[]string) {
	if node.Kind == yaml.MappingNode {
		for index := 0; index+1 < len(node.Content); index += 2 {
			key := node.Content[index]
			value := node.Content[index+1]
			if key.Value == "run" && value.Kind == yaml.ScalarNode {
				*runBlocks = append(*runBlocks, value.Value)
			}
			collectYAMLRunBlocks(value, runBlocks)
		}
		return
	}
	for _, child := range node.Content {
		collectYAMLRunBlocks(child, runBlocks)
	}
}

func commandFields(fields []string, root *cobra.Command, rootFlags, valueFlags map[string]bool) ([]string, error) {
	for len(fields) > 0 && strings.HasPrefix(fields[0], "-") {
		field := fields[0]
		flag := strings.SplitN(field, "=", 2)[0]
		if !rootFlags[flag] {
			return nil, fmt.Errorf("unsupported root flag %q", flag)
		}
		hasInlineValue := strings.Contains(field, "=")
		value := ""
		if hasInlineValue {
			_, value, _ = strings.Cut(field, "=")
		}
		fields = fields[1:]
		if valueFlags[flag] && !hasInlineValue {
			if len(fields) == 0 {
				return nil, fmt.Errorf("flag %q requires a value", flag)
			}
			value = fields[0]
			fields = fields[1:]
		}
		var rootFlag *pflag.Flag
		if strings.HasPrefix(flag, "--") {
			rootFlag = lookupFlag(root, strings.TrimPrefix(flag, "--"))
		} else {
			rootFlag = lookupShorthandFlag(root, strings.TrimPrefix(flag, "-"))
		}
		if err := validateKnownFlagValue(rootFlag, value); err != nil {
			return nil, err
		}
	}
	return fields, nil
}

func rootFlagSets(root *cobra.Command) (map[string]bool, map[string]bool) {
	rootFlags := map[string]bool{}
	valueFlags := map[string]bool{}
	root.PersistentFlags().VisitAll(func(flag *pflag.Flag) {
		longName := "--" + flag.Name
		rootFlags[longName] = true
		if flag.NoOptDefVal == "" {
			valueFlags[longName] = true
		}
		if flag.Shorthand != "" {
			shortName := "-" + flag.Shorthand
			rootFlags[shortName] = true
			if flag.NoOptDefVal == "" {
				valueFlags[shortName] = true
			}
		}
	})
	return rootFlags, valueFlags
}

func commandHelp(t *testing.T, binary string, path []string) string {
	t.Helper()
	args := append(append([]string{}, path...), "--help")
	output, err := exec.Command(binary, args...).CombinedOutput()
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

func resolveCommand(t *testing.T, binary, file string, fields []string, commands map[string]bool, rootFlags map[string]bool, valueFlags map[string]bool) ([]string, string, []string, error) {
	t.Helper()
	if len(fields) == 0 || !commands[fields[0]] {
		return nil, "", nil, fmt.Errorf("unsupported a7 command %q", strings.Join(fields, " "))
	}
	path := []string{fields[0]}
	help := commandHelp(t, binary, path)
	index := 1
	for index < len(fields) {
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
				return path, help, nil, fmt.Errorf("unsupported interspersed flag %q before a7 subcommand", flag)
			}
			index++
			if valueFlags[flag] && !strings.Contains(field, "=") {
				if index >= len(fields) {
					return path, help, nil, fmt.Errorf("flag %q requires a value", flag)
				}
				index++
			}
			continue
		}
		isSubcommand, err := commandFieldIsSubcommand(subcommands, field)
		if err != nil {
			return path, help, nil, fmt.Errorf("a7 %s: %w", strings.Join(path, " "), err)
		}
		if !isSubcommand {
			break
		}
		path = append(path, field)
		help = commandHelp(t, binary, path)
		index++
	}
	return path, help, fields[index:], nil
}

func newA7CommandTree() *cobra.Command {
	ios, _, _, _ := iostreams.Test()
	cfg := config.NewFileConfig()
	factory := &cmd.Factory{
		IOStreams: ios,
		Config: func() (config.Config, error) {
			return cfg, nil
		},
	}
	return rootcmd.NewCmd(factory, cfg)
}

func validatePositionalArgs(root *cobra.Command, path []string, fields []string) error {
	command, remainingPath, err := root.Find(path)
	if err != nil {
		return err
	}
	if len(remainingPath) != 0 {
		return fmt.Errorf("failed to resolve command path %q", strings.Join(path, " "))
	}
	args, err := positionalArgs(command, fields)
	if err != nil {
		return err
	}
	return command.ValidateArgs(args)
}

func positionalArgs(command *cobra.Command, fields []string) ([]string, error) {
	var args []string
	for index := 0; index < len(fields); index++ {
		field := fields[index]
		if isShellControlToken(field) {
			break
		}
		if field == "--" {
			for _, arg := range fields[index+1:] {
				if isShellControlToken(arg) {
					break
				}
				args = append(args, arg)
			}
			break
		}
		if strings.HasPrefix(field, "--") {
			nameValue := strings.TrimPrefix(field, "--")
			name, value, hasInlineValue := strings.Cut(nameValue, "=")
			flag := lookupFlag(command, name)
			if flag == nil {
				return nil, fmt.Errorf("unsupported flag %q", field)
			}
			if !hasInlineValue && flag.NoOptDefVal == "" {
				index++
				if index >= len(fields) {
					return nil, fmt.Errorf("flag %q requires a value", field)
				}
				value = fields[index]
			}
			if err := validateKnownFlagValue(flag, value); err != nil {
				return nil, err
			}
			continue
		}
		if strings.HasPrefix(field, "-") && field != "-" {
			shorthandValue := strings.TrimPrefix(field, "-")
			shorthand := shorthandValue[:1]
			flag := lookupShorthandFlag(command, shorthand)
			if flag == nil {
				return nil, fmt.Errorf("unsupported shorthand flag %q", field)
			}
			hasInlineValue := len(shorthandValue) > 1
			value := strings.TrimPrefix(shorthandValue[1:], "=")
			if !hasInlineValue && flag.NoOptDefVal == "" {
				index++
				if index >= len(fields) {
					return nil, fmt.Errorf("flag %q requires a value", field)
				}
				value = fields[index]
			}
			if err := validateKnownFlagValue(flag, value); err != nil {
				return nil, err
			}
			continue
		}
		args = append(args, field)
	}
	return args, nil
}

func isShellControlToken(field string) bool {
	isAngleBracketPlaceholder := len(field) > 2 && strings.HasPrefix(field, "<") && strings.HasSuffix(field, ">")
	return strings.ContainsAny(field, "|<>") && !isAngleBracketPlaceholder
}

func validateKnownFlagValue(flag *pflag.Flag, value string) error {
	if flag.Name != "output" || value == "" || value == "workflow-expression" || strings.HasPrefix(value, "$") || strings.HasPrefix(value, "<") {
		return nil
	}
	return cmdutil.ValidateOutputFormat(value)
}

func lookupFlag(command *cobra.Command, name string) *pflag.Flag {
	if flag := command.Flags().Lookup(name); flag != nil {
		return flag
	}
	if flag := command.InheritedFlags().Lookup(name); flag != nil {
		return flag
	}
	return command.Root().PersistentFlags().Lookup(name)
}

func lookupShorthandFlag(command *cobra.Command, shorthand string) *pflag.Flag {
	if flag := command.Flags().ShorthandLookup(shorthand); flag != nil {
		return flag
	}
	if flag := command.InheritedFlags().ShorthandLookup(shorthand); flag != nil {
		return flag
	}
	return command.Root().PersistentFlags().ShorthandLookup(shorthand)
}

func shellFields(line string) ([]string, error) {
	var fields []string
	var current strings.Builder
	var quote rune
	var escaped bool
	var started bool
	for _, char := range line {
		if escaped {
			current.WriteRune(char)
			escaped = false
			started = true
			continue
		}
		if quote != 0 {
			if char == quote {
				quote = 0
				continue
			}
			if quote == '"' && char == '\\' {
				escaped = true
				continue
			}
			current.WriteRune(char)
			started = true
			continue
		}
		switch {
		case char == '\\':
			escaped = true
			started = true
		case char == '\'' || char == '"':
			quote = char
			started = true
		case unicode.IsSpace(char):
			if started {
				fields = append(fields, current.String())
				current.Reset()
				started = false
			}
		default:
			current.WriteRune(char)
			started = true
		}
	}
	if escaped {
		return nil, fmt.Errorf("unfinished escape")
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quote %q", string(quote))
	}
	if started {
		fields = append(fields, current.String())
	}
	return fields, nil
}

func cliInvocations(line string, invocationPattern *regexp.Regexp) []string {
	var invocations []string
	for _, match := range invocationPattern.FindAllStringSubmatchIndex(line, -1) {
		if len(match) >= 4 {
			start := match[2]
			end := shellInvocationEnd(line, match[3])
			invocations = append(invocations, strings.TrimSpace(line[start:end]))
		}
	}
	return invocations
}

func shellInvocationEnd(line string, start int) int {
	var quote byte
	var escaped bool
	var substitutionDepth int
	for index := start; index < len(line); index++ {
		current := line[index]
		if escaped {
			escaped = false
			continue
		}
		if quote != '\'' && current == '\\' {
			escaped = true
			continue
		}
		if quote == '\'' {
			if current == '\'' {
				quote = 0
			}
			continue
		}
		if quote == '"' {
			if current == '"' {
				quote = 0
				continue
			}
			if current == '$' && index+1 < len(line) && line[index+1] == '(' {
				substitutionDepth++
				index++
				continue
			}
			if current == ')' && substitutionDepth > 0 {
				substitutionDepth--
			}
			continue
		}
		if quote == '`' {
			if current == '`' {
				quote = 0
			}
			continue
		}

		switch current {
		case '\'', '"', '`':
			quote = current
		case '$':
			if index+1 < len(line) && line[index+1] == '(' {
				substitutionDepth++
				index++
			}
		case ')':
			if substitutionDepth == 0 {
				return index
			}
			substitutionDepth--
		case '|', ';', '&':
			if substitutionDepth == 0 {
				return index
			}
		}
	}
	return len(line)
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
