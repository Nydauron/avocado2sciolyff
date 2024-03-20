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

type Table struct {
	events  []string
	schools []School
}

type School struct {
	teamNumber string
	name       string
	track      string
	scores     []string
	totalScore string
	rank       string
}

func ParseHTML(r io.ReadCloser) Table {
	z := html.NewTokenizer(r)
	table := Table{}
	isTable := false
	isEventName := false
	isTableHead := false
	isTableRow := false
	isTableCell := false
	eventCount := 0
	currentColumn := 0
	bufferSchool := School{}
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
				if t.Data == "th" {
					for _, attr := range t.Attr {
						if attr.Key == "class" {
							classRegex := regexp.MustCompile(`\brotated\b`)
							isEventName = classRegex.MatchString(attr.Val)
						}
					}
				}
				continue
			}
			isTableRow = isTable && t.Data == "tr"
			if isTableRow {
				currentColumn = 0
				bufferSchool = School{}
				continue
			}
			isTableHead = isTable && t.Data == "thead"
			if isTableHead {
				continue
			}
			if t.Data == "table" {
				for _, attr := range t.Attr {
					if attr.Key == "class" {
						classRegex := regexp.MustCompile(`\bresults-table\b`)
						isTable = classRegex.MatchString(attr.Val)
					}
				}
				continue
			}

		case html.TextToken:
			t := z.Token()
			if isTableHead && isEventName {
				table.events = append(table.events, strings.Trim(t.Data, " "))
				eventCount = len(table.events)
				continue
			}
			if !isTableHead && isTableCell {
				trimmedData := strings.Trim(t.Data, " ")
				switch currentColumn {
				case 0:
					bufferSchool.teamNumber = trimmedData
				case 1:
					bufferSchool.name = trimmedData
				case 2:
					bufferSchool.track = trimmedData
				case 2 + eventCount + 1:
					bufferSchool.totalScore = trimmedData
				case 2 + eventCount + 2:
					bufferSchool.rank = trimmedData
				default:
					bufferSchool.scores = append(bufferSchool.scores, trimmedData)
				}
				currentColumn += 1
			}
		case html.EndTagToken:
			t := z.Token()
			if t.Data == "a" || t.Data == "span" {
				isEventName = false
				continue
			}
			if t.Data == "th" || t.Data == "td" {
				isTableCell = false
				continue
			}
			if t.Data == "tr" {
				isTableRow = false
				if bufferSchool.teamNumber != "" && bufferSchool.name != "" {
					table.schools = append(table.schools, bufferSchool)
				}
				bufferSchool = School{}
				currentColumn = 0
				continue
			}
			if t.Data == "table" {
				isTable = false
				continue
			}
		}
	}
}
