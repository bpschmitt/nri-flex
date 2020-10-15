/*
* Copyright 2019 New Relic Corporation. All rights reserved.
* SPDX-License-Identifier: Apache-2.0
 */

package inputs

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Table holds a simple table of headers and rows.
type Table struct {
	Attributes map[string]string
	Headers    []string
	Rows       [][]string
}

// ParseToJSON parses a html fragment or whole document looking for HTML
func ParseToJSON(s []byte) (string, error) {

	tables, err := Parse(s)
	if err != nil {
		return "", err
	}
	jsonString := "["
	closeString := ""
	numberOfTables := len(tables)

	for i, t := range tables {
		if len(t.Rows) == 0 {
			continue
		}
		if i != numberOfTables-1 {
			closeString = ","
		} else {
			closeString = ""
		}
		tString := convertTable(t, i)
		jsonString = jsonString + tString + closeString
	}
	jsonString = jsonString + "]"
	return jsonString, nil
}

func convertTable(t *Table, i int) string {
	j := `{"table":[`

	header := t.Headers
	numberOfRows := len(t.Rows)
	for i, row := range t.Rows {

		j = j + "{"
		numberOfCells := len(row)
		for c := range row {
			r := ""
			if c != numberOfCells-1 {
				r = fmt.Sprintf(" \"%s\": \"%s\",", header[c], row[c])
			} else {
				r = fmt.Sprintf(" \"%s\": \"%s\"", header[c], row[c])
			}

			j = j + r
		}
		if i != numberOfRows-1 {
			j = j + "},"
		} else {
			j = j + "}]"
		}

	}
	for k, v := range t.Attributes {
		kv := fmt.Sprintf(", \"%s\": \"%s\"", k, v)
		j = j + kv

	}
	j = fmt.Sprintf("%s,\"Index\":%d }", j, i)
	return j
}

// Parse parses a html fragment or whole document looking for HTML
// tables. It converts all cells into text, stripping away any HTML content.
func Parse(s []byte) ([]*Table, error) {
	node, err := html.Parse(bytes.NewReader(s))
	if err != nil {
		return nil, err
	}
	tables := []*Table{}
	parse(node, &tables)
	for kk, t := range tables {

		tables[kk] = addMissingColumns(t)
	}

	return tables, nil
}

func innerText(n *html.Node) string {
	if n.Type == html.TextNode {
		stripCR := strings.Replace(n.Data, "\n", "", -1)
		return stripCR
	}
	result := ""
	for x := n.FirstChild; x != nil; x = x.NextSibling {
		result += innerText(x)
	}
	return result
}

func parse(n *html.Node, tables *[]*Table) {
	strip := strings.TrimSpace
	switch n.DataAtom {
	case atom.Table:
		t := &Table{}
		for _, at := range n.Attr {
			if t.Attributes == nil {
				t.Attributes = map[string]string{}
			}
			t.Attributes[at.Key] = at.Val
		}
		*tables = append(*tables, t)
	case atom.Th:
		t := (*tables)[len(*tables)-1]
		t.Headers = append(t.Headers, strip(innerText(n)))
	case atom.Tr:
		t := (*tables)[len(*tables)-1]
		t.Rows = append(t.Rows, []string{})
	case atom.Td:
		t := (*tables)[len(*tables)-1]
		l := len(t.Rows) - 1
		t.Rows[l] = append(t.Rows[l], strip(innerText(n)))
		return
	}
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		parse(child, tables)
	}
}

func addMissingColumns(t *Table) *Table {
	cols := len(t.Headers)
	rows := make([][]string, 0, len(t.Rows))
	for _, row := range t.Rows {
		if len(row) > 0 {
			rows = append(rows, row)
		}
		if len(row) > cols {
			cols = len(row)
		}
	}
	for len(t.Headers) < cols {
		name := "Col " + strconv.Itoa(len(t.Headers)+1)
		t.Headers = append(t.Headers, name)
	}
	for i, row := range t.Headers {
		if len(row) == 0 {
			row = "Col " + strconv.Itoa(i)
		}
		t.Headers[i] = row
	}
	for kk := range rows {
		for len(rows[kk]) < cols {
			rows[kk] = append(rows[kk], "")
		}
	}
	t.Rows = rows
	return t
}
