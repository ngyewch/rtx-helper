package rtx

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	legacyVersionFiles = map[string][]string{
		"crystal":   {".crystal-version"},
		"elixir":    {".exenv-version"},
		"go":        {".go-version", "go.mod"},
		"java":      {".java-version", ".sdkmanrc"},
		"node":      {".nvmrc", ".node-version"},
		"python":    {".python-version"},
		"ruby":      {".ruby-version", "Gemfile"},
		"terraform": {".terraform-version", ".packer-version", "main.tf"},
		"yarn":      {".yarnrc"},
	}
)

type Helper struct {
	DefaultConfigFilename         string
	DefaultToolVersionsFilename   string
	LegacyVersionFile             bool
	LegacyVersionFileDisableTools []string
}

func NewHelper() (*Helper, error) {
	rtxDefaultConfigFilename := os.Getenv("RTX_DEFAULT_CONFIG_FILENAME")
	if rtxDefaultConfigFilename == "" {
		rtxDefaultConfigFilename = ".rtx.toml"
	}

	rtxDefaultToolVersionsFilename := os.Getenv("RTX_DEFAULT_TOOL_VERSIONS_FILENAME")
	if rtxDefaultToolVersionsFilename == "" {
		rtxDefaultToolVersionsFilename = ".tool-versions"
	}

	rtxLegacyVersionFile := os.Getenv("RTX_LEGACY_VERSION_FILE")
	legacyVersionFile := (rtxLegacyVersionFile == "") || (rtxLegacyVersionFile == "1")

	rtxLegacyVersionFileDisableTools := strings.TrimSpace(os.Getenv("RTX_LEGACY_VERSION_FILE_DISABLE_TOOLS"))
	var legacyVersionFileDisableTools []string
	if rtxLegacyVersionFileDisableTools != "" {
		legacyVersionFileDisableTools = strings.Split(rtxLegacyVersionFileDisableTools, ",")
		for i := 0; i < len(legacyVersionFileDisableTools); i++ {
			legacyVersionFileDisableTools[i] = strings.TrimSpace(legacyVersionFileDisableTools[i])
		}
	}

	return &Helper{
		DefaultConfigFilename:         rtxDefaultConfigFilename,
		DefaultToolVersionsFilename:   rtxDefaultToolVersionsFilename,
		LegacyVersionFile:             legacyVersionFile,
		LegacyVersionFileDisableTools: legacyVersionFileDisableTools,
	}, nil
}

func (helper *Helper) HasVersionFiles(path string) (bool, error) {
	f, err := hasFile(path, helper.DefaultConfigFilename)
	if err != nil {
		return false, err
	}
	if f {
		return true, nil
	}

	f, err = hasFile(path, helper.DefaultToolVersionsFilename)
	if err != nil {
		return false, err
	}
	if f {
		return true, nil
	}

	if helper.LegacyVersionFile {
		for toolName, filenames := range legacyVersionFiles {
			disabled := false
			for _, disabledToolName := range helper.LegacyVersionFileDisableTools {
				if disabledToolName == toolName {
					disabled = true
					break
				}
			}
			if disabled {
				continue
			}
			f, err = hasAtLeastOneOfTheListedFiles(path, filenames)
			if err != nil {
				return false, err
			}
			if f {
				return true, nil
			}
		}
	}

	return false, nil
}

func (helper *Helper) ListAvailable(name string) ([]*ToolVersion, error) {
	cmd := exec.Command("rtx", "ls-remote", name)
	buf := bytes.NewBuffer(nil)
	cmd.Stdin = os.Stdin
	cmd.Stdout = buf
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if (cmd.ProcessState != nil) && (cmd.ProcessState.ExitCode() != 0) {
		fmt.Printf("exit code = %d\n", cmd.ProcessState.ExitCode())
	} else {
		if err != nil {
			return nil, err
		}
	}
	var versions []*ToolVersion
	scanner := bufio.NewScanner(buf)
	for scanner.Scan() {
		version := scanner.Text()
		versions = append(versions, NewToolVersion(version))
	}
	return versions, nil
}

func hasFile(path string, filename string) (bool, error) {
	_, err := os.Stat(filepath.Join(path, filename))
	if err == nil {
		return true, nil
	} else if !os.IsNotExist(err) {
		return false, err
	} else {
		return false, nil
	}
}

func hasAtLeastOneOfTheListedFiles(path string, filenames []string) (bool, error) {
	for _, filename := range filenames {
		f, err := hasFile(path, filename)
		if err != nil {
			return false, err
		}
		if f {
			return true, nil
		}
	}
	return false, nil
}
