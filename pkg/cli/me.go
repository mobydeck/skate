package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var meCmd = &cobra.Command{
	Use:   "me",
	Short: "Show the authenticated Mattermost user",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, svc, err := loadConfigAndService()
		if err != nil {
			return err
		}

		me, err := svc.GetMe()
		if err != nil {
			return fmt.Errorf("getting current user: %w", err)
		}

		printStructured(cmd, me, func() {
			fmt.Printf("ID:       %s\n", me.ID)
			fmt.Printf("Username: %s\n", me.Username)
			if name := joinName(me.FirstName, me.LastName); name != "" {
				fmt.Printf("Name:     %s\n", name)
			}
			if me.Nickname != "" {
				fmt.Printf("Nickname: %s\n", me.Nickname)
			}
		})
		return nil
	},
}

func joinName(first, last string) string {
	switch {
	case first != "" && last != "":
		return first + " " + last
	case first != "":
		return first
	case last != "":
		return last
	default:
		return ""
	}
}
