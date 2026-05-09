package cli

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestPrettyFlagHidden verifies the --pretty flag is registered but hidden
// from help output, so it doesn't show up to agents reading `skate --help`.
func TestPrettyFlagHidden(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("pretty")
	if flag == nil {
		t.Fatal("--pretty flag should be registered")
	}
	if !flag.Hidden {
		t.Error("--pretty should be hidden")
	}
	if flag.Value.Type() != "bool" {
		t.Errorf("--pretty type = %s, want bool", flag.Value.Type())
	}
}

// TestPrintMarkdown_PassThroughByDefault verifies that without --pretty, the
// helper writes the input verbatim (no ANSI / no rendering).
func TestPrintMarkdown_PassThroughByDefault(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.PersistentFlags().Bool("pretty", false, "")

	got := captureStdout(t, func() { printMarkdown(cmd, "# Heading\n\nbody\n") })

	if got != "# Heading\n\nbody\n" {
		t.Errorf("unexpected output: %q", got)
	}
}

// TestPrintMarkdown_PrettyDoesNotCrash verifies the --pretty path runs end-to-
// end without panicking and preserves the heading text. We don't assert on
// exact bytes: Glamour's `auto` style detects whether stdout is a real
// terminal and intentionally falls through to passthrough when piped (a real
// user running `skate task ... --pretty | less` shouldn't get ANSI codes).
// The TTY-only render path is exercised manually.
func TestPrintMarkdown_PrettyDoesNotCrash(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.PersistentFlags().Bool("pretty", false, "")
	_ = cmd.PersistentFlags().Set("pretty", "true")

	got := captureStdout(t, func() { printMarkdown(cmd, "# Heading\n\nbody\n") })

	if !strings.Contains(got, "Heading") {
		t.Errorf("output should still contain heading text, got: %q", got)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdout
	os.Stdout = w
	done := make(chan struct{})
	var buf bytes.Buffer
	go func() {
		_, _ = io.Copy(&buf, r)
		close(done)
	}()
	fn()
	w.Close()
	os.Stdout = orig
	<-done
	return buf.String()
}
