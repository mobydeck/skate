package cli

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

//go:embed skills/skate/SKILL.md
var skillFS embed.FS

var setupCmd = &cobra.Command{
	Use:   "setup <agent>",
	Short: "Register skate MCP server with an AI agent",
	Long:  "Supported agents: claude-code (claude), cursor, codex, opencode, roocode (roo)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agent := strings.ToLower(args[0])
		project, _ := cmd.Flags().GetBool("project")

		switch agent {
		case "claude", "claude-code":
			return setupClaude(project)
		case "cursor":
			return setupCursor(project)
		case "codex":
			return setupCodex()
		case "opencode":
			return setupOpenCode(project)
		case "roo", "roocode":
			return setupRooCode()
		default:
			return fmt.Errorf("unknown agent %q. Supported: claude-code, cursor, codex, opencode, roocode", agent)
		}
	},
}

func init() {
	setupCmd.Flags().BoolP("project", "p", false, "Install for current project only")
}

func mcpEntry() map[string]interface{} {
	return map[string]interface{}{
		"type":    "stdio",
		"command": "skate",
		"args":    []string{"mcp"},
		"env":     map[string]string{},
	}
}

func setupClaude(project bool) error {
	var configPath string
	if project {
		configPath = ".mcp.json"
	} else {
		home, _ := os.UserHomeDir()
		configPath = filepath.Join(home, ".claude.json")
	}

	if err := upsertMCPConfig(configPath, "mcpServers", "skate"); err != nil {
		return err
	}
	fmt.Printf("Claude Code: MCP registered in %s\n", configPath)

	if !project {
		if err := installSkill("claude"); err != nil {
			return err
		}
	}
	return nil
}

func setupCursor(project bool) error {
	var configPath string
	if project {
		configPath = filepath.Join(".cursor", "mcp.json")
	} else {
		home, _ := os.UserHomeDir()
		configPath = filepath.Join(home, ".cursor", "mcp.json")
	}

	if err := upsertMCPConfig(configPath, "mcpServers", "skate"); err != nil {
		return err
	}
	fmt.Printf("Cursor: MCP registered in %s\n", configPath)

	if err := installSkill("cursor"); err != nil {
		return err
	}
	return nil
}

func setupCodex() error {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".codex", "config.json")

	if err := upsertMCPConfig(configPath, "mcpServers", "skate"); err != nil {
		return err
	}
	fmt.Printf("Codex: MCP registered in %s\n", configPath)

	if err := installSkill("codex"); err != nil {
		return err
	}
	return nil
}

func setupOpenCode(project bool) error {
	var configPath string
	if project {
		configPath = "opencode.json"
	} else {
		home, _ := os.UserHomeDir()
		configPath = filepath.Join(home, ".config", "opencode", "opencode.json")
	}

	if err := upsertMCPConfig(configPath, "mcp", "skate"); err != nil {
		return err
	}
	fmt.Printf("OpenCode: MCP registered in %s\n", configPath)
	return nil
}

func setupRooCode() error {
	configPath := filepath.Join(".roo", "mcp.json")

	if err := upsertMCPConfig(configPath, "mcpServers", "skate"); err != nil {
		return err
	}
	fmt.Printf("RooCode: MCP registered in %s\n", configPath)
	return nil
}

func upsertMCPConfig(path, serversKey, toolName string) error {
	dir := filepath.Dir(path)
	if dir != "." {
		os.MkdirAll(dir, 0o755)
	}

	var data map[string]interface{}
	raw, err := os.ReadFile(path)
	if err == nil {
		json.Unmarshal(raw, &data)
	}
	if data == nil {
		data = make(map[string]interface{})
	}

	servers, ok := data[serversKey].(map[string]interface{})
	if !ok {
		servers = make(map[string]interface{})
	}

	servers[toolName] = mcpEntry()
	data[serversKey] = servers

	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	return os.WriteFile(path, out, 0o644)
}

func installSkill(agent string) error {
	home, _ := os.UserHomeDir()

	var skillDir string
	switch agent {
	case "claude":
		skillDir = filepath.Join(home, ".claude", "skills", "skate")
	case "cursor":
		skillDir = filepath.Join(home, ".cursor", "skills", "skate")
	case "codex":
		skillDir = filepath.Join(home, ".codex", "skills", "skate")
	default:
		return nil
	}

	os.MkdirAll(skillDir, 0o755)
	content, err := skillFS.ReadFile("skills/skate/SKILL.md")
	if err != nil {
		return fmt.Errorf("reading embedded skill: %w", err)
	}

	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillPath, content, 0o644); err != nil {
		return fmt.Errorf("writing skill: %w", err)
	}
	fmt.Printf("Skill installed: %s\n", skillPath)
	return nil
}
