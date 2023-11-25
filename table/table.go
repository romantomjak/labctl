package table

import (
	"fmt"
	"io"
)

type Table struct {
	header            []string
	data              [][]string
	columnWidths      []int
	columnMarginRight int
}

func New(columns ...string) *Table {
	t := &Table{
		header:            columns,
		data:              make([][]string, 0),
		columnWidths:      make([]int, 0, len(columns)),
		columnMarginRight: 2,
	}
	for _, c := range columns {
		t.columnWidths = append(t.columnWidths, len(c))
	}
	return t
}

func (t *Table) AddRow(columns ...string) error {
	t.data = append(t.data, columns)
	for i, col := range columns {
		if t.columnWidths[i] < len(col) {
			t.columnWidths[i] = len(col) + t.columnMarginRight
		}
	}
	return nil
}

func (t *Table) Print(w io.Writer) error {
	if len(t.data) == 0 {
		return nil
	}

	for i, col := range t.header {
		format := fmt.Sprintf("%%-%ds", t.columnWidths[i])
		_, err := fmt.Fprintf(w, format, col)
		if err != nil {
			return err
		}
	}

	fmt.Println()

	for _, row := range t.data {
		for i, col := range row {
			format := fmt.Sprintf("%%-%ds", t.columnWidths[i])
			_, err := fmt.Fprintf(w, format, col)
			if err != nil {
				return err
			}
		}
		fmt.Println()
	}

	return nil
}
