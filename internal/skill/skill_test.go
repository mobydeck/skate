package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRaw_HasFrontmatter(t *testing.T) {
	if !strings.HasPrefix(string(Raw()), "---\n") {
		t.Fatalf("Raw() should preserve YAML frontmatter for the agent skill loader")
	}
	if !strings.Contains(string(Raw()), "name: skate") {
		t.Errorf("frontmatter should contain skill name")
	}
}

func TestMarkdown_StripsFrontmatter(t *testing.T) {
	md := Markdown()
	if strings.HasPrefix(md, "---") {
		t.Errorf("Markdown() should not start with frontmatter delimiter, got: %q", md[:20])
	}
	if strings.Contains(md, "name: skate\n") {
		t.Errorf("Markdown() should not contain frontmatter fields")
	}
	// Sanity: still has the body content.
	if !strings.Contains(md, "Skate") {
		t.Error("Markdown() should contain the skill body")
	}
}

func TestStripFrontmatter(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "strips frontmatter and leading blank lines",
			in:   "---\nname: x\ndesc: y\n---\n\n# Heading\nbody\n",
			want: "# Heading\nbody\n",
		},
		{
			name: "passes through if no frontmatter",
			in:   "# Heading\nbody\n",
			want: "# Heading\nbody\n",
		},
		{
			name: "passes through if frontmatter not closed",
			in:   "---\nname: x\nbody without close\n",
			want: "---\nname: x\nbody without close\n",
		},
		{
			name: "empty input",
			in:   "",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stripFrontmatter(tt.in); got != tt.want {
				t.Errorf("stripFrontmatter(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestPointer_FallbackWhenNoSkillFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	got := Pointer()
	if !strings.Contains(got, "skate_help") {
		t.Errorf("expected skate_help fallback when no skill installed, got: %s", got)
	}
}

func TestPointer_PointsAtInstalledFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	skillPath := filepath.Join(tmp, ".claude", "skills", "skate", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(skillPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(skillPath, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := Pointer()
	if !strings.Contains(got, skillPath) {
		t.Errorf("expected pointer to reference %s, got: %s", skillPath, got)
	}
}

func TestPointer_ResolutionOrder(t *testing.T) {
	// When both Claude Code and Cursor paths exist, Claude wins (first in order).
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	claudePath := filepath.Join(tmp, ".claude", "skills", "skate", "SKILL.md")
	cursorPath := filepath.Join(tmp, ".cursor", "skills", "skate", "SKILL.md")
	for _, p := range []string{claudePath, cursorPath} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	got := Pointer()
	if !strings.Contains(got, claudePath) {
		t.Errorf("expected Claude path to win, got: %s", got)
	}
	if strings.Contains(got, cursorPath) {
		t.Errorf("Cursor path should not appear when Claude path exists, got: %s", got)
	}
}
