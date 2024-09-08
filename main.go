package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/urfave/cli/v2"
	"golang.org/x/net/html"
	"gopkg.in/yaml.v3"
)

func cliHandle(inputLocation string, outputWriter io.Writer) error {
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
		defer resp.Body.Close()
		contentType := resp.Header.Get("content-type")
		expectedContent := "text/html; charset=UTF-8"
		if contentType != expectedContent {
			fmt.Fprintf(os.Stderr, "Page content recieved is not text/html UTF-8. Got instead \"%s\n", contentType)
		}
		htmlBodyReader = resp.Body
	} else if f, err := os.Open(inputLocation); err == nil {
		fmt.Fprintln(os.Stderr, "File detected")
		defer f.Close()
		htmlBodyReader = f
	} else {
		return fmt.Errorf("provided input was neither a valid URL or a path to existing file: %v", inputLocation)
	}

	table, err := parseHTML(htmlBodyReader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cell did not contain number: %v\n", err)
		os.Exit(4)
		return nil
	}

	sciolyffDump := generateSciolyFF(*table)

	yamlEncoder := yaml.NewEncoder(outputWriter)
	yamlEncoder.SetIndent(2)
	err = yamlEncoder.Encode(&sciolyffDump)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Encoding to YAML failed: %v", err)
		os.Exit(3)
		return nil
	}

	err = yamlEncoder.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Encoding to YAML failed on close: %v", err)
		os.Exit(3)
		return nil
	}

	return nil
}

const (
	inputFlag     = "input"
	outputFlag    = "output"
	stdoutCLIName = "-"
)

func main() {
	var inputLocation string
	var outputLocation string = ""
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
			&cli.StringFlag{
				Name:        outputFlag,
				Aliases:     []string{"o"},
				Usage:       "The location to write the YAML result. Can be a file path or \"-\" (for stdout).",
				Required:    true,
				Destination: &outputLocation,
			},
		},
		Action: func(cCtx *cli.Context) error {
			if outputLocation == "" {
				return fmt.Errorf("output not set")
			}
			var outputWriter io.WriteCloser = os.Stdout
			if outputLocation != stdoutCLIName {
				var err error
				outputWriter, err = os.OpenFile(outputLocation, os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					return fmt.Errorf("could not create or open file: %v", err)
				}
			}
			return cliHandle(inputLocation, outputWriter)
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

type sciolyFF struct {
	Events   []event   `yaml:"Events"`
	Placings []placing `yaml:"Placings"`
}

type table struct {
	events  []string
	schools []school
}

type tournamentMetadata struct {
	Name      string `yaml:"name"`
	ShortName string `yaml:"short name"`
	Location  string `yaml:"location"`
	Level     string `yaml:"level"`
	State     string `yaml:"state"`
	Division  string `yaml:"division"`
	Year      string `yaml:"year"`
	Date      string `yaml:"date"`
}

type event struct {
	Name               string `yaml:"name"`
	IsTrial            bool   `yaml:"trial"`
	TrialedNormalEvent bool   `yaml:"trialed"`
	ScoringObjective   string `yaml:"scoring,omitempty"`
}

type placing struct {
	Event        string `yaml:"event"`
	TeamNumber   string `yaml:"team"`
	Participated bool   `yaml:"participated"`
	EventDQ      bool   `yaml:"disqualified"`
	Exempt       bool   `yaml:"exempt"`
	Unknown      bool   `yaml:"unknown"`
	Tie          bool   `yaml:"tie"`
	Place        uint   `yaml:"place"`
}

type school struct {
	TeamNumber string `yaml:"number"`
	Name       string `yaml:"school"`
	Track      string `yaml:"track"`
	Scores     []uint `yaml:"-"`
	TotalScore string `yaml:"-"`
	Rank       string `yaml:"-"`
}

func parseHTML(r io.ReadCloser) (*table, error) {
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
			return &table, nil
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
					bufferSchool.TeamNumber = trimmedData
				case 1:
					bufferSchool.Name = trimmedData
				case 2:
					bufferSchool.Track = trimmedData
				case 2 + eventCount + 1:
					bufferSchool.TotalScore = trimmedData
				case 2 + eventCount + 2:
					bufferSchool.Rank = trimmedData
				default:
					place, err := strconv.ParseUint(trimmedData, 10, 8)
					if err != nil {
						return nil, err
					}
					bufferSchool.Scores = append(bufferSchool.Scores, uint(place))
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
				if bufferSchool.TeamNumber != "" && bufferSchool.Name != "" {
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

func generateSciolyFF(table table) sciolyFF {
	events := make([]event, 0)
	for _, eventName := range table.events {
		events = append(events, event{Name: eventName, IsTrial: false, TrialedNormalEvent: false})
	}

	placings := make([]placing, 0)
	teamCount := uint(len(table.schools))
	for _, team := range table.schools {
		for eventIdx, score := range team.Scores {
			p := placing{Event: events[eventIdx].Name, TeamNumber: team.TeamNumber}
			switch {
			case score == teamCount: // P
				p.Participated = true
			case score >= teamCount+2: // DQ
				p.EventDQ = true
			}
			p.Place = score
			placings = append(placings, p)
		}
	}

	return sciolyFF{Events: events, Placings: placings}
}
