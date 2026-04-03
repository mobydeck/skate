package boards

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// TextTranslator translates text to English. Nil means no translation.
type TextTranslator interface {
	Translate(text string) string
}

func RenderCardMarkdown(card *Card, board *Board, blocks []*Block, summaries []*TimeEntrySummary, uc *UserCache, tr TextTranslator, maxComments ...int) string {
	defs := ParsePropertyDefs(board)
	var sb strings.Builder

	// Title
	icon := card.Icon
	if icon != "" {
		icon += " "
	}
	sb.WriteString(fmt.Sprintf("# %s%s\n\n", icon, tl(tr, card.Title)))

	// Properties table
	sb.WriteString("| Property | Value |\n|----------|-------|\n")
	for _, d := range defs {
		val, ok := card.Properties[d.ID]
		if !ok || val == nil {
			continue
		}
		resolved := ResolvePropertyValue(defs, d.ID, val)
		if resolved == "" {
			continue
		}
		// Resolve person properties to usernames
		if (d.Type == "person" || d.Type == "multiPerson") && uc != nil {
			resolved = uc.Resolve(resolved)
		}
		sb.WriteString(fmt.Sprintf("| %s | %s |\n", d.Name, resolved))
	}
	sb.WriteString("\n")

	// Content blocks
	var contentBlocks, comments, attachments []*Block
	for _, b := range blocks {
		switch b.Type {
		case "text", "divider", "checkbox", "h1", "h2", "h3", "image":
			contentBlocks = append(contentBlocks, b)
		case "comment":
			comments = append(comments, b)
		case "attachment":
			attachments = append(attachments, b)
		}
	}

	if len(contentBlocks) > 0 {
		sb.WriteString("## Description\n\n")
		for _, b := range contentBlocks {
			switch b.Type {
			case "h1":
				sb.WriteString("# " + tl(tr, b.Title) + "\n\n")
			case "h2":
				sb.WriteString("## " + tl(tr, b.Title) + "\n\n")
			case "h3":
				sb.WriteString("### " + tl(tr, b.Title) + "\n\n")
			case "divider":
				sb.WriteString("---\n\n")
			case "checkbox":
				checked := ""
				if v, ok := b.Fields["value"]; ok && v == true {
					checked = "x"
				}
				sb.WriteString(fmt.Sprintf("- [%s] %s\n", checked, tl(tr, b.Title)))
			case "image":
				fileID := ""
				if fid, ok := b.Fields["fileId"]; ok {
					fileID = fmt.Sprintf("%v", fid)
				}
				name := b.Title
				if name == "" {
					name = fileID
				}
				sb.WriteString(fmt.Sprintf("![%s](fileId: %s)\n\n", name, fileID))
			default:
				sb.WriteString(tl(tr, b.Title) + "\n\n")
			}
		}
	}

	sort.Slice(comments, func(i, j int) bool {
		return comments[i].CreateAt < comments[j].CreateAt
	})

	if len(comments) > 0 {
		limit := 0
		if len(maxComments) > 0 && maxComments[0] > 0 {
			limit = maxComments[0]
		}

		shown := comments
		hidden := 0
		if limit > 0 && len(comments) > limit {
			hidden = len(comments) - limit
			shown = comments[hidden:] // show last N (most recent)
		}

		sb.WriteString("## Comments\n\n")
		if hidden > 0 {
			sb.WriteString(fmt.Sprintf("*(%d earlier comments not shown, use --full to see all)*\n\n", hidden))
		}
		for _, c := range shown {
			date := FormatTimestamp(c.CreateAt)
			author := c.CreatedBy
			if uc != nil {
				author = uc.Resolve(c.CreatedBy)
			}
			sb.WriteString(fmt.Sprintf("**@%s** (%s):\n> %s\n\n", author, date, tl(tr, c.Title)))
		}
	}

	if len(attachments) > 0 {
		sb.WriteString("## Attachments\n\n")
		for _, a := range attachments {
			fileID := ""
			if fid, ok := a.Fields["fileId"]; ok {
				fileID = fmt.Sprintf("%v", fid)
			}
			name := a.Title
			if name == "" {
				name = fileID
			}
			sb.WriteString(fmt.Sprintf("- %s (type: %s, fileId: %s)\n", name, a.Type, fileID))
		}
		sb.WriteString("\n")
	}

	if len(summaries) > 0 {
		sb.WriteString("## Time Tracking\n\n")
		var totalSeconds int64
		for _, ts := range summaries {
			userName := ts.UserID
			if uc != nil {
				userName = uc.Resolve(ts.UserID)
			}

			if ts.RunningEntry != nil {
				elapsed := computeElapsed(ts.RunningEntry.StartTime)
				sb.WriteString(fmt.Sprintf("- @%s: %s + %s running\n", userName, ts.TotalDisplay, FormatDuration(elapsed)))
				totalSeconds += ts.TotalSeconds + elapsed
			} else {
				sb.WriteString(fmt.Sprintf("- @%s: %s\n", userName, ts.TotalDisplay))
				totalSeconds += ts.TotalSeconds
			}
		}
		sb.WriteString(fmt.Sprintf("\nTotal: %s\n", FormatDuration(totalSeconds)))
	}

	return sb.String()
}

func RenderComments(blocks []*Block, uc *UserCache, tr TextTranslator) string {
	var comments []*Block
	for _, b := range blocks {
		if b.Type == "comment" {
			comments = append(comments, b)
		}
	}

	if len(comments) == 0 {
		return "No comments.\n"
	}

	sort.Slice(comments, func(i, j int) bool {
		return comments[i].CreateAt < comments[j].CreateAt
	})

	var sb strings.Builder
	for _, c := range comments {
		date := FormatTimestamp(c.CreateAt)
		author := c.CreatedBy
		if uc != nil {
			author = uc.Resolve(c.CreatedBy)
		}
		sb.WriteString(fmt.Sprintf("**@%s** (%s):\n> %s\n\n", author, date, tl(tr, c.Title)))
	}
	return sb.String()
}

func tl(tr TextTranslator, text string) string {
	if tr == nil {
		return text
	}
	return tr.Translate(text)
}

func computeElapsed(startTimeMs int64) int64 {
	if startTimeMs == 0 {
		return 0
	}
	elapsed := time.Now().UnixMilli() - startTimeMs
	if elapsed < 0 {
		return 0
	}
	return elapsed / 1000
}
