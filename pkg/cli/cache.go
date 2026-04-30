package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"skate/internal/config"
)

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Inspect and clean Skate's local cache",
	Long: `Skate's cache lives at ~/.cache/skate (Linux) / %LocalAppData%\skate (Windows).
It currently holds:
  users.yaml             — username/ID lookup cache
  downloads/             — files saved by skate_download when no output_path was passed`,
}

var cacheLsCmd = &cobra.Command{
	Use:     "ls",
	Aliases: []string{"list"},
	Short:   "List cached download files",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := config.DownloadsDir()
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("No cached downloads (%s does not exist).\n", dir)
				return nil
			}
			return fmt.Errorf("reading %s: %w", dir, err)
		}

		type row struct {
			name    string
			size    int64
			modTime time.Time
		}
		var rows []row
		var total int64
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			rows = append(rows, row{e.Name(), info.Size(), info.ModTime()})
			total += info.Size()
		}

		if len(rows) == 0 {
			fmt.Printf("No cached downloads in %s.\n", dir)
			return nil
		}

		sort.Slice(rows, func(i, j int) bool { return rows[i].modTime.After(rows[j].modTime) })

		fmt.Printf("Cache: %s\n\n", dir)
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "FILE\tSIZE\tMODIFIED")
		for _, r := range rows {
			fmt.Fprintf(w, "%s\t%s\t%s\n", r.name, humanSize(r.size), r.modTime.Format("2006-01-02 15:04"))
		}
		w.Flush()
		fmt.Printf("\n%d files, %s total\n", len(rows), humanSize(total))
		return nil
	},
}

var cacheCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Delete all cached download files",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := config.DownloadsDir()
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("Cache already empty.")
				return nil
			}
			return fmt.Errorf("reading %s: %w", dir, err)
		}

		var removed int
		var freed int64
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			info, _ := e.Info()
			path := filepath.Join(dir, e.Name())
			if err := os.Remove(path); err != nil {
				fmt.Fprintf(os.Stderr, "  failed: %s: %v\n", e.Name(), err)
				continue
			}
			removed++
			if info != nil {
				freed += info.Size()
			}
		}
		if removed == 0 {
			fmt.Println("Cache already empty.")
		} else {
			fmt.Printf("Removed %d file(s), freed %s from %s\n", removed, humanSize(freed), dir)
		}
		return nil
	},
}

func humanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

func init() {
	cacheCmd.AddCommand(cacheLsCmd)
	cacheCmd.AddCommand(cacheCleanCmd)
}
