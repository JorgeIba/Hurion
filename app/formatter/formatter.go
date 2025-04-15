package formatter

import (
	"fmt"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

func FormatToMarkdownTable(title string, headers []any, rows [][]any) string {
	var sb strings.Builder

	t := table.NewWriter()
	t.SetOutputMirror(&sb)
	t.SetStyle(table.StyleRounded)
	t.Style().Format.Header = text.FormatTitle

	t.AppendHeader(table.Row(headers))
	for _, row := range rows {
		t.AppendRow(table.Row(row))
	}

	// Center columns
	columnConfigs := make([]table.ColumnConfig, len(headers))
	for i := range headers {
		columnConfigs[i] = table.ColumnConfig{
			Number: i + 1,
			Align:  text.AlignCenter,
		}
	}
	t.SetColumnConfigs(columnConfigs)

	t.Render()
	return fmt.Sprintf("```markdown\n%s\n\n%s```", title, sb.String())
}
