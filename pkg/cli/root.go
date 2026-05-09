package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/charmbracelet/glamour"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"

	"skate/internal/boards"
	"skate/internal/client"
	"skate/internal/config"
	"skate/internal/version"
)

var rootCmd = &cobra.Command{
	Use:     "skate",
	Short:   "Skate - access Mattermost Boards from CLI and MCP",
	Version: version.Version,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolP("json", "j", false, "Output in JSON format")
	rootCmd.PersistentFlags().BoolP("yaml", "y", false, "Output in YAML format")
	// Hidden, undocumented: pretty-print markdown output via Glamour. Strictly a
	// human-UX nicety — agents and scripts should not rely on it (the rendered
	// output is ANSI-escaped and unsuitable for parsing).
	rootCmd.PersistentFlags().BoolP("pretty", "P", false, "")
	_ = rootCmd.PersistentFlags().MarkHidden("pretty")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(localInitCmd)
	rootCmd.AddCommand(boardsCmd)
	rootCmd.AddCommand(tasksCmd)
	rootCmd.AddCommand(taskCmd)
	rootCmd.AddCommand(taskFilesCmd)
	rootCmd.AddCommand(downloadCmd)
	rootCmd.AddCommand(updateStatusCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(commentCmd)
	rootCmd.AddCommand(addContentCmd)
	rootCmd.AddCommand(attachCmd)
	rootCmd.AddCommand(timerStartCmd)
	rootCmd.AddCommand(timerStopCmd)
	rootCmd.AddCommand(timeAddCmd)
	rootCmd.AddCommand(mcpCmd)
	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(promptCmd)
	rootCmd.AddCommand(commentsCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(findCmd)
	rootCmd.AddCommand(statusesCmd)
	rootCmd.AddCommand(meCmd)
	rootCmd.AddCommand(deleteBlockCmd)
	rootCmd.AddCommand(usersCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(editBlockCmd)
	rootCmd.AddCommand(nextCmd)
	rootCmd.AddCommand(stateCmd)
	rootCmd.AddCommand(skillCmd)
	rootCmd.AddCommand(cacheCmd)
}

func loadConfigAndService() (*config.Config, *boards.Service, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, fmt.Errorf("loading config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, nil, err
	}
	c := client.New(config.BaseURL(cfg), cfg.Token)
	svc := boards.NewService(c, cfg.TeamID)
	return cfg, svc, nil
}

type outputFormat int

const (
	formatDefault outputFormat = iota
	formatJSON
	formatYAML
)

func getOutputFormat(cmd *cobra.Command) outputFormat {
	if j, _ := cmd.Flags().GetBool("json"); j {
		return formatJSON
	}
	if y, _ := cmd.Flags().GetBool("yaml"); y {
		return formatYAML
	}
	return formatDefault
}

// printMarkdown writes md to stdout. When the hidden --pretty flag is set, it
// runs the input through Glamour for terminal rendering. On any rendering
// failure it falls back to the raw text so output is never lost.
func printMarkdown(cmd *cobra.Command, md string) {
	pretty, _ := cmd.Flags().GetBool("pretty")
	if !pretty {
		fmt.Print(md)
		return
	}
	out, err := glamour.Render(md, "auto")
	if err != nil {
		fmt.Print(md)
		return
	}
	fmt.Print(out)
}

func printStructured(cmd *cobra.Command, data any, defaultFn func()) {
	switch getOutputFormat(cmd) {
	case formatJSON:
		out, _ := json.MarshalIndent(data, "", "  ")
		fmt.Println(string(out))
	case formatYAML:
		out, _ := yaml.Marshal(data)
		fmt.Print(string(out))
	default:
		defaultFn()
	}
}

func requireBoardID(cfg *config.Config, flagBoardID string) (string, error) {
	if flagBoardID != "" {
		return flagBoardID, nil
	}
	if cfg.BoardID != "" {
		return cfg.BoardID, nil
	}
	return "", fmt.Errorf("board ID required: use --board flag or run 'skate local-init'")
}
