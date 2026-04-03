package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"skate/internal/boards"
	"skate/internal/client"
	"skate/internal/config"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize or update global skate configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		reader := bufio.NewReader(os.Stdin)

		// Load existing config if present
		existing, _ := config.LoadGlobal()
		if existing == nil {
			existing = &config.Config{}
		}

		urlStr := prompt(reader, "Mattermost URL", existing.MattermostURL)
		token := prompt(reader, "Personal access token", maskToken(existing.Token))
		// If user kept the masked token, use the original
		if token == maskToken(existing.Token) {
			token = existing.Token
		}

		// Validate connection
		c := client.New(urlStr+"/plugins/focalboard/api/v2", token)
		svc := boards.NewService(c, "")
		user, err := svc.GetMe()
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
		name := strings.TrimSpace(user.FirstName + " " + user.LastName)
		if name == "" {
			name = user.Username
		}
		fmt.Printf("Connected as: %s (@%s)\n\n", name, user.Username)

		// Team selection
		teamID := promptTeam(reader, c, existing.TeamID)
		if teamID == "" {
			return fmt.Errorf("team ID is required")
		}

		// Preserve existing translate config, or set defaults on first run
		translate := existing.Translate
		if translate.Provider == "" && translate.Model == "" {
			translate = config.DefaultTranslateConfig()
		}

		cfg := &config.Config{
			MattermostURL: urlStr,
			Token:         token,
			TeamID:        teamID,
			OnlyMine:      existing.OnlyMine,
			Translate:     translate,
		}

		path := config.GlobalConfigPath()
		if err := cfg.Save(path); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}
		fmt.Printf("\nConfig saved to %s\n", path)
		return nil
	},
}

func prompt(reader *bufio.Reader, label, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", label, defaultVal)
	} else {
		fmt.Printf("%s: ", label)
	}
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal
	}
	return input
}

func maskToken(token string) string {
	if len(token) <= 8 {
		return token
	}
	return token[:4] + "..." + token[len(token)-4:]
}

func promptTeam(reader *bufio.Reader, c *client.Client, defaultTeamID string) string {
	teamData, err := c.Get("/teams")
	if err != nil {
		return prompt(reader, "Team ID", defaultTeamID)
	}

	var teams []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	if json.Unmarshal(teamData, &teams) != nil || len(teams) == 0 {
		return prompt(reader, "Team ID", defaultTeamID)
	}

	fmt.Println("Available teams:")
	defaultIdx := -1
	for i, t := range teams {
		title := t.Title
		if title == "" {
			title = "(default)"
		}
		marker := "  "
		if t.ID == defaultTeamID {
			marker = "* "
			defaultIdx = i
		}
		fmt.Printf("  %s%d. %s — %s\n", marker, i+1, title, t.ID)
	}
	fmt.Println()

	if len(teams) == 1 && defaultTeamID == "" {
		fmt.Printf("Auto-selected: %s\n", teams[0].Title)
		return teams[0].ID
	}

	defaultHint := ""
	if defaultIdx >= 0 {
		defaultHint = fmt.Sprintf("%d", defaultIdx+1)
	}
	input := prompt(reader, "Enter team number or ID", defaultHint)

	// Try as number
	for i, t := range teams {
		if fmt.Sprintf("%d", i+1) == input {
			return t.ID
		}
	}
	// Try as ID
	for _, t := range teams {
		if t.ID == input {
			return t.ID
		}
	}
	// If input was empty and we had a default
	if input == defaultHint && defaultIdx >= 0 {
		return teams[defaultIdx].ID
	}

	return defaultTeamID
}
