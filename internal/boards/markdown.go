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
	fmt.Fprintf(&sb, "# %s%s\n\n", icon, tl(tr, card.Title))

	// Properties table
	sb.WriteString("| Property | Value |\n|----------|-------|\n")
	for _, d := range defs {
		val, ok := card.Properties[d.ID]
		if !ok || val == nil {
			continue
		}
		var resolved string
		if d.Type == "person" || d.Type == "multiPerson" {
			resolved = resolvePersonValue(val, uc)
			// Only @-prefix when names were resolved through UserCache; raw IDs
			// are kept as-is so callers without a cache see the unmodified value.
			if resolved != "" && uc != nil {
				parts := strings.Split(resolved, ", ")
				for i, p := range parts {
					parts[i] = "@" + p
				}
				resolved = strings.Join(parts, ", ")
			}
		} else {
			resolved = ResolvePropertyValue(defs, d.ID, val)
		}
		if resolved == "" {
			continue
		}
		fmt.Fprintf(&sb, "| %s | %s |\n", d.Name, resolved)
	}
	if card.CreatedBy != "" {
		creator := card.CreatedBy
		if uc != nil {
			creator = uc.Resolve(card.CreatedBy)
		}
		fmt.Fprintf(&sb, "| Created By | @%s |\n", creator)
	}
	// Show available statuses so agents know valid values
	if statusProp := FindPropertyByName(defs, "Status"); statusProp != nil {
		var vals []string
		for _, o := range statusProp.Options {
			vals = append(vals, o.Value)
		}
		fmt.Fprintf(&sb, "| Available Statuses | %s |\n", strings.Join(vals, ", "))
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

	// Order content blocks per card.ContentOrder (the authoritative order shown
	// in the web UI). Blocks not referenced in ContentOrder fall to the end in
	// their original arrival order.
	orderIdx := flattenContentOrder(card.ContentOrder)
	sort.SliceStable(contentBlocks, func(i, j int) bool {
		pi, oki := orderIdx[contentBlocks[i].ID]
		pj, okj := orderIdx[contentBlocks[j].ID]
		if oki && okj {
			return pi < pj
		}
		return oki && !okj
	})

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
				fmt.Fprintf(&sb, "- [%s] %s\n", checked, tl(tr, b.Title))
			case "image":
				fileID := ""
				if fid, ok := b.Fields["fileId"]; ok {
					fileID = fmt.Sprintf("%v", fid)
				}
				name := b.Title
				if name == "" {
					name = fileID
				}
				fmt.Fprintf(&sb, "![%s](fileId: %s)\n\n", name, fileID)
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
			fmt.Fprintf(&sb, "*(%d earlier comments not shown, use --full to see all)*\n\n", hidden)
		}
		for _, c := range shown {
			date := FormatTimestamp(c.CreateAt)
			author := c.CreatedBy
			if uc != nil {
				author = uc.Resolve(c.CreatedBy)
			}
			fmt.Fprintf(&sb, "**@%s** (%s):\n> %s\n\n", author, date, tl(tr, c.Title))
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
			fmt.Fprintf(&sb, "- %s (type: %s, fileId: %s)\n", name, a.Type, fileID)
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
				fmt.Fprintf(&sb, "- @%s: %s + %s running\n", userName, ts.TotalDisplay, FormatDuration(elapsed))
				totalSeconds += ts.TotalSeconds + elapsed
			} else {
				fmt.Fprintf(&sb, "- @%s: %s\n", userName, ts.TotalDisplay)
				totalSeconds += ts.TotalSeconds
			}
		}
		fmt.Fprintf(&sb, "\nTotal: %s\n", FormatDuration(totalSeconds))
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
		fmt.Fprintf(&sb, "**@%s** (%s):\n> %s\n\n", author, date, tl(tr, c.Title))
	}
	return sb.String()
}

// flattenContentOrder turns a card's ContentOrder into a {blockID: position}
// map. Focalboard stores the order as a list whose elements are either a
// string (block ID) or an array of strings (a row of grouped blocks).
func flattenContentOrder(order []any) map[string]int {
	idx := make(map[string]int, len(order))
	pos := 0
	for _, e := range order {
		switch v := e.(type) {
		case string:
			idx[v] = pos
			pos++
		case []any:
			for _, inner := range v {
				if s, ok := inner.(string); ok {
					idx[s] = pos
					pos++
				}
			}
		case []string:
			for _, s := range v {
				idx[s] = pos
				pos++
			}
		}
	}
	return idx
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
