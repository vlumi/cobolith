package layout

import (
	"strconv"
	"strings"

	"github.com/vlumi/cobolith/cobol"
)

// ReportLines lays out the lines a report prints as the records of its
// file: one for each LINE of each report group, its columns placed where
// their COLUMN clauses put them, as wide as their pictures. A line is
// named after its group, by the group's name or else its TYPE and
// control, PAGE-HEADING or CONTROL-FOOTING-FINAL, and numbered in order
// when the group has more than one, PAGE-HEADING-LINE-2. A column with no COLUMN
// clause follows the one before it, and one with no name is named after
// its line and its column, PAGE-HEADING-COL-70, since what it prints is
// worth following to.
func ReportLines(rd cobol.ReportDescription, dialect cobol.Dialect) ([]*Field, error) {
	groups, err := buildTree(rd.Entries)
	if err != nil {
		return nil, err
	}
	var records []*Field
	for _, group := range groups {
		lines := reportLines(group)
		for i, line := range lines {
			record := &Field{Level: 1, Name: groupName(group), Comment: group.Comment, Pos: line.Pos}
			if len(lines) > 1 {
				record.Name += "-LINE-" + strconv.Itoa(i+1)
			}
			end := 0
			for _, column := range reportColumns(line) {
				if column.Start == 0 {
					column.Start = end + 1
				}
				if column.Usage == "" {
					column.Usage = "DISPLAY"
				}
				if err := column.placeElementary(sign{}, dialect); err != nil {
					return nil, err
				}
				end = max(end, column.Start+column.Length-1)
				if column.Name == "FILLER" {
					column.Name = record.Name + "-COL-" + strconv.Itoa(column.Start)
				}
				record.Children = append(record.Children, column)
			}
			record.Start, record.Length = 1, end
			records = append(records, record)
		}
	}
	return records, nil
}

// groupName names a report group: its own name, or its TYPE and control
// joined by hyphens.
func groupName(group *Field) string {
	if group.Name != "FILLER" {
		return group.Name
	}
	name := group.reportType
	if group.reportControl != "" {
		name += " " + group.reportControl
	}
	return strings.ReplaceAll(name, " ", "-")
}

// reportLines lists the entries of a group that carry a LINE clause: the
// group itself when it does, else those below it.
func reportLines(group *Field) []*Field {
	if group.reportLine != "" {
		return []*Field{group}
	}
	var lines []*Field
	var walk func(f *Field)
	walk = func(f *Field) {
		for _, child := range f.Children {
			if child.reportLine != "" {
				lines = append(lines, child)
			} else {
				walk(child)
			}
		}
	}
	walk(group)
	return lines
}

// reportColumns lists the elementary entries of a line, in order: each a
// column with a picture, placed at its COLUMN.
func reportColumns(line *Field) []*Field {
	var columns []*Field
	var walk func(f *Field)
	walk = func(f *Field) {
		for _, child := range f.Children {
			if len(child.Children) > 0 {
				walk(child)
				continue
			}
			if child.Picture == "" {
				continue
			}
			column := *child
			column.Children = nil
			column.Start = child.reportColumn
			columns = append(columns, &column)
		}
	}
	walk(line)
	if line.Picture != "" && len(line.Children) == 0 {
		column := *line
		column.Start = line.reportColumn
		columns = append(columns, &column)
	}
	return columns
}
