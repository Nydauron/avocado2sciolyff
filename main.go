package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

func main() {
	if len(os.Args) <= 1 {
		fmt.Fprintln(os.Stderr, "Please provide a URL")
		os.Exit(1)
		return
	}
	rawUrl := os.Args[1]

	resp, err := http.Get(rawUrl)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error occurred when trying to fetch page: %v\n", err)
		os.Exit(2)
		return
	}

	contentType := resp.Header.Get("content-type")
	expectedContent := "text/html; charset=UTF-8"
	if contentType != expectedContent {
		fmt.Fprintf(os.Stderr, "Page content recieved is not text/html UTF-8. Got instead \"%s\n", contentType)
	}
	table := ParseHTML(resp.Body)
	fmt.Printf("%v", table)
}

type TableRow struct {
	cells []TableCell
}

type TableCell struct {
	data string
}

func ParseHTML(r io.ReadCloser) []TableRow {
	z := html.NewTokenizer(r)
	table := make([]TableRow, 0)
	isTable := false
	isTableRow := false
	isTableCell := false
	bufferRow := TableRow{}
	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			return table
		case html.StartTagToken:
			t := z.Token()
			switch t.Data {
			case "span":
				fallthrough
			case "a":
				continue
			}
			isTableCell = isTableRow && (t.Data == "th" || t.Data == "td")
			if isTableCell {
				continue
			}
			isTableRow = isTable && t.Data == "tr"
			if isTableRow {
				bufferRow = TableRow{}
				continue
			}
			if t.Data == "table" {
				for _, attr := range t.Attr {
					if attr.Key == "class" {
						classRegex := regexp.MustCompile(`\btable-striped\b`)
						isTable = classRegex.MatchString(attr.Val)
					}
				}
				continue
			}

		case html.TextToken:
			t := z.Token()
			if isTableCell {
				bufferRow.cells = append(bufferRow.cells, TableCell{data: strings.Trim(t.Data, " ")})
			}
		case html.EndTagToken:
			t := z.Token()
			if t.Data == "th" || t.Data == "td" {
				isTableCell = false
				continue
			}
			if t.Data == "tr" {
				isTableRow = false
				table = append(table, bufferRow)
				bufferRow = TableRow{}
				continue
			}
			if t.Data == "table" {
				isTable = false
				continue
			}
		}
	}
}
