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
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Nydauron/avogado-to-sciolyff/parsers"
	"github.com/Nydauron/avogado-to-sciolyff/sciolyff"
	"github.com/Nydauron/avogado-to-sciolyff/writers"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

const (
	eventIsTrial    = true
	eventWasTrialed = false

	inputFlag     = "input"
	outputFlag    = "output"
	csvFlag       = "csv"
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

func cliHandle(inputLocation string, outputWriter io.Writer, isCSVFile bool) error {
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

	var table *parsers.Table
	if isCSVFile {
		var err error
		table, err = parsers.ParseCSV(htmlBodyReader)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Cell did not contain number: %v\n", err)
			os.Exit(4)
			return nil
		}
	} else {
		var err error
		table, err = parsers.ParseHTML(htmlBodyReader)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Cell did not contain number: %v\n", err)
			os.Exit(4)
			return nil
		}
	}

	sciolyffDump := generateSciolyFF(*table)

	yamlEncoder := yaml.NewEncoder(outputWriter)
	yamlEncoder.SetIndent(2)
	err := yamlEncoder.Encode(&sciolyffDump)
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
	var outputLocation = ""
	var isCSV = false
	app := &cli.App{
		Name:    "avocado2sciolyff",
		Usage:   "A tool to turn table results on Avogadro to sciolyff results",
		Version: semanticVersion,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        csvFlag,
				Usage:       "File passed in is a CSV rather than an HTML file",
				Destination: &isCSV,
			},
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
				outputWriter = writers.NewLazyWriteCloser(func() (io.WriteCloser, error) {
					return os.OpenFile(outputLocation, os.O_CREATE|os.O_WRONLY, 0644)
				})
			}
			return cliHandle(inputLocation, outputWriter, isCSV)
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func generateSciolyFF(table parsers.Table) sciolyff.SciolyFF {
	events := make([]sciolyff.Event, 0)
	for _, e := range table.Events {
		isEventTrialEvent := false
		if e.IsMarkedAsTrial {
			isEventTrialEvent = eventDistingushTrialMarkerPrompt(e.Name)
		}
		events = append(events, sciolyff.Event{Name: e.Name, IsTrial: e.IsMarkedAsTrial && isEventTrialEvent, TrialedNormalEvent: e.IsMarkedAsTrial && !isEventTrialEvent})
	}

	placings := make([]*sciolyff.Placing, 0)
	teamCount := uint(len(table.Schools))
	trackNames := map[string]struct{}{}
	for _, team := range table.Schools {
		trackNames[team.Track] = struct{}{}
		for eventIdx, score := range team.Scores {
			p := sciolyff.Placing{Event: events[eventIdx].Name, TeamNumber: team.TeamNumber}
			p.Participated = true
			if score >= teamCount+1 { // NS
				p.Participated = false
			}
			if score >= teamCount+2 { // DQ
				p.EventDQ = true
			}
			p.Place = score
			placings = append(placings, p)
		}
	}

	tracks := []sciolyff.Track{}

	for trackName := range trackNames {
		tracks = append(tracks, sciolyff.Track{Name: trackName})
	}

	tournament := sciolyff.TournamentMetadata{
		Name:      prompt("Tournament name: "),
		ShortName: prompt("Tournament nickname/short name: "),
		Location:  prompt("Tournament location (host building/campus): "),
		Level:     tournamentLevelPrompt(),
		State:     statePrompt(),
		Division:  tournamentDivisionPrompt(),
		Year:      rulesYearPrompt(),
		Date:      tournamentDatePrompt(),
	}

	return sciolyff.SciolyFF{Tournament: tournament, Tracks: tracks, Events: events, Teams: table.Schools, Placings: placings}
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
