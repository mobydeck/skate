package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var usersCmd = &cobra.Command{
	Use:   "users [QUERY]",
	Short: "List team members (filterable by username substring)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, svc, err := loadConfigAndService()
		if err != nil {
			return err
		}

		users, err := svc.ListUsers(cfg.TeamID)
		if err != nil {
			return fmt.Errorf("listing users: %w", err)
		}

		if len(args) == 1 {
			q := strings.ToLower(args[0])
			filtered := users[:0]
			for _, u := range users {
				if strings.Contains(strings.ToLower(u.Username), q) ||
					strings.Contains(strings.ToLower(u.FirstName+" "+u.LastName), q) {
					filtered = append(filtered, u)
				}
			}
			users = filtered
		}

		sort.Slice(users, func(i, j int) bool { return users[i].Username < users[j].Username })

		if len(users) == 0 {
			fmt.Println("No users found.")
			return nil
		}

		printStructured(cmd, users, func() {
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tUSERNAME\tNAME")
			for _, u := range users {
				fmt.Fprintf(w, "%s\t%s\t%s\n", u.ID, u.Username, joinName(u.FirstName, u.LastName))
			}
			w.Flush()
		})
		return nil
	},
}
