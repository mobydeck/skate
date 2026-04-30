// Package skill exposes the canonical Skate workflow guide (SKILL.md) and
// helpers for telling agents how to find it. The same content is shipped two
// ways: written to disk by `skate setup <agent>`, and returned inline by the
// `skate_help` MCP tool.
package skill

import (
	_ "embed"
	"os"
	"path/filepath"
	"strings"
)

//go:embed SKILL.md
var raw string

// Markdown returns the SKILL.md body with YAML frontmatter stripped. This is
// what we want to surface to a model as a workflow guide — the frontmatter
// (name + description) is only meaningful to the on-disk skill loader.
func Markdown() string {
	return stripFrontmatter(raw)
}

// Raw returns the SKILL.md file contents verbatim, including frontmatter.
// Use this when writing to disk so the agent's skill loader can parse it.
func Raw() []byte {
	return []byte(raw)
}

// Pointer returns a one-line directive for the MCP `initialize` handshake
// telling the connected agent how to load the workflow guide. Resolution
// order:
//  1. If a SKILL.md is installed at one of the well-known agent paths
//     (Claude Code, Cursor, Codex), point at the file so the agent can read
//     it directly with its filesystem tools.
//  2. Otherwise instruct the agent to call the `skate_help` MCP tool, which
//     returns the same content inline.
func Pointer() string {
	if path := installedPath(); path != "" {
		return "Before using any skate_* tool, read " + path + " — it contains the Skate workflow guide (mention rules, status conventions, plan-attachment policy). Do this once per session."
	}
	return "Before using any skate_* tool, call the skate_help tool to load the Skate workflow guide. Do this once per session."
}

// installedPath returns the first existing skill file across the supported
// agent install paths. Empty string if none are present.
func installedPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	for _, sub := range [][]string{
		{".claude", "skills", "skate", "SKILL.md"},
		{".cursor", "skills", "skate", "SKILL.md"},
		{".codex", "skills", "skate", "SKILL.md"},
	} {
		p := filepath.Join(append([]string{home}, sub...)...)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// stripFrontmatter removes a leading YAML frontmatter block (delimited by
// `---` lines). Returns the input unchanged if no frontmatter is present.
func stripFrontmatter(s string) string {
	if !strings.HasPrefix(s, "---\n") {
		return s
	}
	rest := s[len("---\n"):]
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		return s
	}
	return strings.TrimLeft(rest[end+len("\n---\n"):], "\n")
}
