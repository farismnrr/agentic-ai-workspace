package gitstatus

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	maxGitConfigBytes  = 1 << 20
	maxConfigListBytes = 2 << 20
)

type safeGitConfig struct {
	fileMode          string
	ignoreCase        string
	precomposeUnicode string
	symlinks          string
	objectFormat      string
	repositoryFormat  int
}

func preflightGitConfig(snapshot *metadataSnapshot) (safeGitConfig, error) {
	output, err := runGit(snapshot, maxConfigListBytes+1,
		"config", "--local", "--no-includes", "--null", "--list")
	if err != nil {
		return safeGitConfig{}, fmt.Errorf("preflight Git config: %w", err)
	}
	if len(output) > maxConfigListBytes {
		return safeGitConfig{}, errGitConfigUnsupported
	}
	return parseSafeGitConfig(output)
}

func parseSafeGitConfig(output []byte) (safeGitConfig, error) {
	config := safeGitConfig{objectFormat: "sha1"}
	for _, item := range strings.Split(string(output), "\x00") {
		if item == "" {
			continue
		}
		key, value, ok := splitConfigItem(item)
		if !ok {
			return safeGitConfig{}, errGitConfigUnsupported
		}
		key = strings.ToLower(key)
		if forbiddenGitConfigKey(key) {
			return safeGitConfig{}, errGitConfigUnsafe
		}
		if err := keepSafeGitConfig(&config, key, value); err != nil {
			return safeGitConfig{}, err
		}
	}
	if config.objectFormat == "sha256" && config.repositoryFormat == 0 {
		return safeGitConfig{}, errGitObjectFormat
	}
	return config, nil
}

func splitConfigItem(item string) (string, string, bool) {
	index := strings.IndexAny(item, "\n=")
	if index <= 0 {
		return "", "", false
	}
	return item[:index], item[index+1:], true
}

func forbiddenGitConfigKey(key string) bool {
	return indirectGitConfigKey(key) || commandGitConfigKey(key)
}

func indirectGitConfigKey(key string) bool {
	if strings.HasPrefix(key, "filter.") || strings.HasPrefix(key, "include.") ||
		strings.HasPrefix(key, "includeif.") {
		return true
	}
	switch key {
	case "core.worktree", "core.fsmonitor", "core.hookspath", "core.attributesfile",
		"core.excludesfile", "extensions.worktreeconfig", "extensions.partialclone":
		return true
	default:
		return false
	}
}

func commandGitConfigKey(key string) bool {
	return key == "core.sshcommand" || key == "core.askpass" || key == "diff.external" ||
		key == "interactive.difffilter" || strings.HasPrefix(key, "diff.") && strings.HasSuffix(key, ".command") ||
		strings.HasPrefix(key, "merge.") && strings.HasSuffix(key, ".driver") ||
		strings.HasPrefix(key, "credential.") && strings.HasSuffix(key, ".helper")
}

func keepSafeGitConfig(config *safeGitConfig, key, value string) error {
	switch key {
	case "core.repositoryformatversion":
		return setRepositoryFormat(config, value)
	case "extensions.objectformat":
		return setObjectFormat(config, value)
	default:
		return keepSafeBooleanConfig(config, key, value)
	}
}

func setRepositoryFormat(config *safeGitConfig, value string) error {
	version, err := strconv.Atoi(value)
	if err != nil || version < 0 || version > 1 {
		return errGitConfigUnsupported
	}
	config.repositoryFormat = version
	return nil
}

func setObjectFormat(config *safeGitConfig, value string) error {
	if value != "sha1" && value != "sha256" {
		return errGitObjectFormat
	}
	config.objectFormat = value
	return nil
}

func keepSafeBooleanConfig(config *safeGitConfig, key, value string) error {
	target := configBooleanTarget(config, key)
	if target == nil {
		return nil
	}
	return setConfigBoolean(target, value)
}

func configBooleanTarget(config *safeGitConfig, key string) *string {
	switch key {
	case "core.filemode":
		return &config.fileMode
	case "core.ignorecase":
		return &config.ignoreCase
	case "core.precomposeunicode":
		return &config.precomposeUnicode
	case "core.symlinks":
		return &config.symlinks
	default:
		return nil
	}
}

func setConfigBoolean(target *string, value string) error {
	parsed, ok := gitBoolean(value)
	if !ok {
		return errGitConfigUnsupported
	}
	*target = strconv.FormatBool(parsed)
	return nil
}

func gitBoolean(value string) (bool, bool) {
	switch strings.ToLower(value) {
	case "", "true", "yes", "on", "1":
		return true, true
	case "false", "no", "off", "0":
		return false, true
	default:
		return false, false
	}
}

func (config safeGitConfig) contents() []byte {
	var contents strings.Builder
	fmt.Fprintf(&contents, "[core]\n\trepositoryformatversion = %d\n", config.repositoryFormat)
	writeBoolean(&contents, "filemode", config.fileMode)
	writeBoolean(&contents, "ignorecase", config.ignoreCase)
	writeBoolean(&contents, "precomposeunicode", config.precomposeUnicode)
	writeBoolean(&contents, "symlinks", config.symlinks)
	contents.WriteString("\tbare = false\n\tfsmonitor = false\n")
	contents.WriteString("\thooksPath = /dev/null\n\tattributesFile = /dev/null\n")
	contents.WriteString("\texcludesFile = /dev/null\n\n[status]\n\trenames = true\n")
	if config.objectFormat == "sha256" {
		contents.WriteString("\n[extensions]\n\tobjectFormat = sha256\n")
	}
	return []byte(contents.String())
}

func writeBoolean(contents *strings.Builder, key, value string) {
	if value == "" {
		return
	}
	fmt.Fprintf(contents, "\t%s = %s\n", key, value)
}
