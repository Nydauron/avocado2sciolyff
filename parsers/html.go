package parsers

import (
	"io"
	"regexp"
	"strconv"
	"strings"

	sciolyff_models "github.com/Nydauron/avocado2sciolyff/sciolyff/models"
	"golang.org/x/net/html"
)

func ParseHTML(r io.ReadCloser) (*Table, error) {
	z := html.NewTokenizer(r)
	table := Table{}
	isTable := false

	isTournamentInfo := false
	isProcessingEventHeader := false
	isEventName := false
	isEventPotentialTrial := false

	isTableHead := false
	isTableRow := false
	isTableCell := false
	eventCount := 0
	currentColumn := 0
	bufferSchool := sciolyff_models.School{}
	bufferEvent := AvogadroEvent{}
	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			return &table, nil
		case html.StartTagToken:
			t := z.Token()
			switch t.Data {
			case "h1":
				// Tournament info
				if isProcessingEventHeader || isTable || isTableHead || isTableRow || isTableCell {
					continue
				}
				isTournamentInfo = true
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
			isTableRowTag := t.Data == "tr"
			if isTableRowTag {
				for _, attr := range t.Attr {
					if attr.Key == "class" {
						classRegex := regexp.MustCompile(`\bseparator\b`)
						isTableRowTag = !classRegex.MatchString(attr.Val)
					}
				}
			}
			isTableRow = isTable && isTableRowTag
			if isTableRow {
				currentColumn = 0
				bufferSchool = sciolyff_models.School{}
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
			if isTournamentInfo {
				// Parse tournament info

				state_split := strings.SplitN(trimmedData, ":", 2)
				state := ""
				rem := trimmedData
				if len(state_split) == 2 {
					state = strings.Trim(state_split[0], " ")
					rem = strings.Trim(state_split[1], " ")
				}
				if strings.ToLower(state) == "invitational" {
					state = ""
				}

				division_str := strings.TrimFunc(rem, func(r rune) bool {
					return !strings.ContainsRune("()", r)
				})
				division, found := strings.CutPrefix(strings.ToLower(division_str), "div")
				division = strings.Trim(division, ". ")
				if !found {
					division = ""
				}

				left_div_parentheses := strings.Index(rem, "(")
				name := strings.Trim(rem[:left_div_parentheses], " ")

				info := AvogadroTournamentInfo{
					Name:     name,
					Division: division,
					State:    state,
				}
				table.Info = info
			}
			if isEventName {
				bufferEvent.Name = trimmedData
				continue
			}
			if isEventPotentialTrial {
				if strings.Contains(trimmedData, "Trial") {
					bufferEvent.IsMarkedAsTrial = true
				}
			}
			if !isTableHead && isTableCell {
				switch currentColumn {
				case 0:
					teamNumber, err := strconv.ParseUint(numberRegex.FindString(trimmedData), 10, 16)
					if err != nil {
						return nil, err
					}
					bufferSchool.TeamNumber = uint(teamNumber)
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
			if t.Data == "h1" {
				isTournamentInfo = false
				continue
			}
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
					table.Events = append(table.Events, bufferEvent)
					eventCount = len(table.Events)
					bufferEvent = AvogadroEvent{}
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
				if bufferSchool.TeamNumber != 0 && bufferSchool.Name != "" {
					table.Schools = append(table.Schools, bufferSchool)
				}
				bufferSchool = sciolyff_models.School{}
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
