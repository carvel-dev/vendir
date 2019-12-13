package main

import (
	"github.com/cppforlife/go-cli-ui/ui"
	uitbl "github.com/cppforlife/go-cli-ui/ui/table"
)

type NullLogger struct{}

var _ ui.ExternalLogger = NullLogger{}

func (l NullLogger) Error(tag, msg string, args ...interface{}) {}
func (l NullLogger) Debug(tag, msg string, args ...interface{}) {}

func main() {
	ui := ui.NewConfUI(NullLogger{})

	table := uitbl.Table{
		Content: "stemcells",

		Header: []uitbl.Header{
			uitbl.NewHeader("Name"),
			uitbl.NewHeader("Version"),
			uitbl.NewHeader("OS"),
			uitbl.NewHeader("CPI"),
			uitbl.NewHeader("CID"),
		},

		SortBy: []uitbl.ColumnSort{
			{Column: 0, Asc: true},
			{Column: 1, Asc: false},
		},

		Notes: []string{"(*) Currently deployed"},
	}

	stemcells := []struct{}{}

	for _, _ = range stemcells {
		table.Rows = append(table.Rows, []uitbl.Value{
			uitbl.NewValueString("name"),
			uitbl.NewValueSuffix(
				uitbl.NewValueString("version"),
				"*",
			),
			uitbl.NewValueString("name"),
			uitbl.NewValueString("cpi"),
			uitbl.NewValueString("cid"),
		})
	}

	ui.PrintTable(table)
}
