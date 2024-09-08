package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/urfave/cli/v2"
	"golang.org/x/net/html"
)

func cliHandle(inputLocation string) error {
	var htmlBodyReader io.ReadCloser
	if u, err := url.ParseRequestURI(inputLocation); err == nil {
		fmt.Fprintln(os.Stderr, "URL detected")
		rawURL := u.String()
		resp, err := http.Get(rawURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error occurred when trying to fetch page: %v\n", err)
			os.Exit(2)
			return nil
		}

		if resp.StatusCode >= 400 {
			return fmt.Errorf("invalid HTTP status code received: %v", resp.Status)
		}
		contentType := resp.Header.Get("content-type")
		expectedContent := "text/html; charset=UTF-8"
		if contentType != expectedContent {
			fmt.Fprintf(os.Stderr, "Page content recieved is not text/html UTF-8. Got instead \"%s\n", contentType)
		}
		htmlBodyReader = resp.Body
	} else if f, err := os.Open(inputLocation); err == nil {
		fmt.Fprintln(os.Stderr, "File detected")
		htmlBodyReader = f
	} else {
		return fmt.Errorf("provided input was neither a valid URL or a path to existing file: %v", inputLocation)
	}

	table := parseHTML(htmlBodyReader)
	fmt.Printf("%v", table)

	return nil
}

const (
	inputFlag = "input"
)

func main() {
	var inputLocation string
	app := &cli.App{
		Name:  "avocado2sciolyff",
		Usage: "A tool to turn table results on Avogadro to sciolyff results",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        inputFlag,
				Aliases:     []string{"i"},
				Usage:       "The URL or path to the HTML file containing the table to convert",
				Destination: &inputLocation,
				Required:    true,
			},
		},
		Action: func(cCtx *cli.Context) error {
			return cliHandle(inputLocation)
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

type table struct {
	events  []string
	schools []school
}

type school struct {
	teamNumber string
	name       string
	track      string
	scores     []string
	totalScore string
	rank       string
}

func parseHTML(r io.ReadCloser) table {
	z := html.NewTokenizer(r)
	table := table{}
	isTable := false
	isEventName := false
	isTableHead := false
	isTableRow := false
	isTableCell := false
	eventCount := 0
	currentColumn := 0
	bufferSchool := school{}
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
				bufferSchool = school{}
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
				currentColumn++
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
				bufferSchool = school{}
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
