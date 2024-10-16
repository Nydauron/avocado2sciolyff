package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"maps"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
	"golang.org/x/net/html"
	"gopkg.in/yaml.v3"
)

const (
	eventIsTrial    = true
	eventWasTrialed = false

	inputFlag     = "input"
	outputFlag    = "output"
	stdoutCLIName = "-"
)

var build string
var semanticVersion = "v0.1.0-dev" + build

var stateMapping = map[string]string{
	"ALABAMA":              "AL",
	"ALASKA":               "AK",
	"ARIZONA":              "AZ",
	"ARKANSAS":             "AR",
	"NORTH CALIFORNIA":     "NCA",
	"SOUTH CALIFORNIA":     "SCA",
	"COLORADO":             "CO",
	"CONNECTICUT":          "CT",
	"DELAWARE":             "DE",
	"DISTRICT OF COLUMBIA": "DC",
	"FLORIDA":              "FL",
	"GEORGIA":              "GA",
	"HAWAII":               "HI",
	"IDAHO":                "ID",
	"ILLINOIS":             "IL",
	"INDIANA":              "IN",
	"IOWA":                 "IA",
	"KANSAS":               "KS",
	"KENTUCKY":             "KY",
	"LOUISIANA":            "LA",
	"MAINE":                "ME",
	"MARYLAND":             "MD",
	"MASSACHUSETS":         "MA",
	"MICHIGAN":             "MI",
	"MINNESOTA":            "MN",
	"MISSISSIPPI":          "MS",
	"MISSOURI":             "MO",
	"MONTANA":              "MT",
	"NEBRASKA":             "NE",
	"NEVADA":               "NV",
	"NEW HAMPSHIRE":        "NH",
	"NEW JERSEY":           "NJ",
	"NEW MEXICO":           "NM",
	"NEW YORK":             "NY",
	"NORTH CAROLINA":       "NC",
	"NORTH DAKOTA":         "ND",
	"OHIO":                 "OH",
	"OKLAHOMA":             "OK",
	"OREGON":               "OR",
	"PENNSYLVANIA":         "PA",
	"ROAD ISLAND":          "RI",
	"SOUTH CAROLINA":       "SC",
	"SOUTH DAKOTA":         "SD",
	"TENNESSEE":            "TN",
	"TEXAS":                "TX",
	"UTAH":                 "UT",
	"VERMONT":              "VT",
	"VIRGINIA":             "VA",
	"WASHINGTON":           "WA",
	"WEST VIRGINIA":        "WV",
	"WISCONSIN":            "WI",
	"WYOMING":              "WY",
}

var stateAbbreviations = func() []string {
	arr := make([]string, 0, len(stateMapping))
	for v := range maps.Values(stateMapping) {
		arr = append(arr, v)
	}

	return arr
}()

var stateNames = func() []string {
	arr := make([]string, 0, len(stateMapping))
	for v := range maps.Keys(stateMapping) {
		arr = append(arr, v)
	}

	return arr
}()

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

func main() {
	var inputLocation string
	var outputLocation string = ""
	app := &cli.App{
		Name:    "avocado2sciolyff",
		Usage:   "A tool to turn table results on Avogadro to sciolyff results",
		Version: semanticVersion,
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

type table struct {
	events  []avogadroEvent
	schools []school
}

type avogadroEvent struct {
	name            string
	isMarkedAsTrial bool
}

type sciolyFF struct {
	Tournament tournamentMetadata `yaml:"Tournament"`
	Tracks     []track            `yaml:"Tracks,omitempty"`
	Events     []event            `yaml:"Events"`
	Teams      []school           `yaml:"Teams"`
	Placings   []placing          `yaml:"Placings"`
}

type track struct {
	Name string `yaml:"name"`
}

type tournamentMetadata struct {
	Name      string `yaml:"name"`
	ShortName string `yaml:"short name,omitempty"`
	Location  string `yaml:"location"`
	Level     string `yaml:"level"`
	State     string `yaml:"state"`
	Division  string `yaml:"division"`
	Year      int    `yaml:"year"`
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

	isProcessingEventHeader := false
	isEventName := false
	isEventPotentialTrial := false

	isTableHead := false
	isTableRow := false
	isTableCell := false
	eventCount := 0
	currentColumn := 0
	bufferSchool := school{}
	bufferEvent := avogadroEvent{}
	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			return &table, nil
		case html.StartTagToken:
			t := z.Token()
			switch t.Data {
			case "span":
				if isProcessingEventHeader {
					for _, attr := range t.Attr {
						if attr.Key == "class" {
							classLabelWarningRegex := regexp.MustCompile(`\blabel-warning\b`)
							isEventPotentialTrial = classLabelWarningRegex.MatchString(attr.Val)
						}
					}
				}
				continue
			case "a":
				if isProcessingEventHeader {
					isEventName = true
				}
				continue
			}
			isTableCell = isTableRow && (t.Data == "th" || t.Data == "td")
			if isTableCell {
				if t.Data == "th" {
					for _, attr := range t.Attr {
						if attr.Key == "class" {
							classRegex := regexp.MustCompile(`\brotated\b`)
							isProcessingEventHeader = classRegex.MatchString(attr.Val)
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
			trimmedData := strings.Trim(t.Data, " ")
			if isEventName {
				bufferEvent.name = trimmedData
				continue
			}
			if isEventPotentialTrial {
				if strings.Contains(trimmedData, "Trial") {
					bufferEvent.isMarkedAsTrial = true
				}
			}
			if !isTableHead && isTableCell {
				switch currentColumn {
				case 0:
					bufferSchool.TeamNumber = trimmedData
				case 1:
					bufferSchool.Name = trimmedData
				case 2:
					bufferSchool.Track = strings.Trim(trimmedData, "()")
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
			if t.Data == "a" {
				isEventName = false
				continue
			}
			if t.Data == "span" {
				isEventPotentialTrial = false
				continue
			}
			if t.Data == "th" {
				if isProcessingEventHeader && isTableHead {
					table.events = append(table.events, bufferEvent)
					eventCount = len(table.events)
					bufferEvent = avogadroEvent{}
				}
				isProcessingEventHeader = false
				isEventPotentialTrial = false
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
	for _, e := range table.events {
		isEventTrialEvent := false
		if e.isMarkedAsTrial {
			isEventTrialEvent = eventDistingushTrialMarkerPrompt(e.name)
		}
		events = append(events, event{Name: e.name, IsTrial: e.isMarkedAsTrial && isEventTrialEvent, TrialedNormalEvent: e.isMarkedAsTrial && !isEventTrialEvent})
	}

	placings := make([]placing, 0)
	teamCount := uint(len(table.schools))
	trackNames := map[string]struct{}{}
	for _, team := range table.schools {
		trackNames[team.Track] = struct{}{}
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

	tracks := []track{}

	for trackName := range trackNames {
		tracks = append(tracks, track{Name: trackName})
	}

	tournament := tournamentMetadata{
		Name:      prompt("Tournament name: "),
		ShortName: prompt("Tournament nickname/short name: "),
		Location:  prompt("Tournament location (host building/campus): "),
		Level:     tournamentLevelPrompt(),
		State:     statePrompt(),
		Division:  tournamentDivisionPrompt(),
		Year:      rulesYearPrompt(),
		Date:      tournamentDatePrompt(),
	}

	return sciolyFF{Tournament: tournament, Tracks: tracks, Events: events, Teams: table.schools, Placings: placings}
}

func eventDistingushTrialMarkerPrompt(eventName string) bool {
	for {
		userInput := prompt(fmt.Sprintf("Event %s had a trial marker. Was this event a trial event (1) or was the event trialed (2)? ", eventName))
		if userSelection, err := strconv.ParseInt(userInput, 10, 8); err == nil {
			switch userSelection {
			case 1:
				return eventIsTrial
			case 2:
				return eventWasTrialed
			}
		}
	}
}

func tournamentDatePrompt() string {
	for {
		userInput := prompt("Tournament date: ")
		_, err := time.Parse(time.DateOnly, userInput)
		if err == nil {
			return userInput
		}
	}
}

func rulesYearPrompt() int {
	for {
		userInput := prompt("Rules Year: ")
		parsedYear, err := strconv.Atoi(userInput)
		if err == nil {
			return parsedYear
		}
	}
}

func tournamentDivisionPrompt() string {
	translatedDivision := ""
	for translatedDivision == "" {
		userInput := strings.ToUpper(prompt("Tournament division (a, b, c): "))
		if userInput[0] >= 'A' || userInput[0] <= 'C' {
			translatedDivision = userInput[:1]
		}
	}
	return translatedDivision
}

func tournamentLevelPrompt() string {
	translatedLevel := ""
	for translatedLevel == "" {
		userInput := prompt("Tournament level (i, r, s, n): ")
		translatedLevel = translateLevelAbbrevToFull(strings.ToLower(userInput)[0])
	}
	return translatedLevel
}

func statePrompt() string {
	translatedState := ""
	for translatedState == "" {
		userInput := strings.ToUpper(prompt("State: "))
		if slices.Contains(stateAbbreviations, userInput) {
			translatedState = userInput
		} else if slices.Contains(stateNames, userInput) {
			translatedState = stateMapping[userInput]
		}
	}
	return translatedState
}

func translateLevelAbbrevToFull(a byte) string {
	switch a {
	case 'i':
		return "Invitational"
	case 'r':
		return "Regionals"
	case 's':
		return "States"
	case 'n':
		return "Nationals"
	default:
		return ""
	}
}

func prompt(message string) string {
	fmt.Fprint(os.Stderr, message)
	buf := bufio.NewReader(os.Stdin)
	input, _ := buf.ReadString('\n')
	return strings.TrimRight(input, "\n")
}
