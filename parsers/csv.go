package parsers

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	sciolyff_models "github.com/Nydauron/avocado2sciolyff/sciolyff/models"
)

const TEAM_NUMER_COL_NAME = ""
const SCHOOL_COL_NAME = "School"
const TOTAL_COL_NAME = "Total"
const PLACE_COL_NAME = "Place"
const TRIAL_MARKER = "Trial"

func ParseCSV(r io.ReadCloser) (*Table, error) {
	buf := bufio.NewReader(r)
	column_str, err := buf.ReadString('\n')
	if err != nil {
		return nil, err
	}
	columns := strings.Split(strings.TrimRight(column_str, "\r\n"), ",")
	columnCount := len(columns)

	avogadroEvents := []AvogadroEvent{}
	for _, colName := range columns {
		if colName != TEAM_NUMER_COL_NAME && colName != SCHOOL_COL_NAME && colName != TOTAL_COL_NAME && colName != PLACE_COL_NAME {
			eventName, hasTrialMarker := strings.CutSuffix(colName, TRIAL_MARKER)
			event := AvogadroEvent{
				Name:            strings.Trim(eventName, " "),
				IsMarkedAsTrial: hasTrialMarker,
			}
			avogadroEvents = append(avogadroEvents, event)
		}
	}

	parsedTable := Table{
		Events:  avogadroEvents,
		Schools: []sciolyff_models.School{},
	}
	for {
		school := sciolyff_models.School{}
		row, err := buf.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, err
		}
		if row != "" {
			cells := strings.Split(strings.TrimRight(row, "\r\n"), ",")
			cellCount := len(cells)
			if cellCount != columnCount {
				return nil, fmt.Errorf("row has different amount of cells than the number of expected column headers: %v", cellCount)
			}
			for i, col := range columns {
				trimmedCell := strings.Trim(cells[i], " ")
				switch col {
				case TEAM_NUMER_COL_NAME:
					// Team number
					teamNumber, err := strconv.ParseUint(numberRegex.FindString(trimmedCell), 10, 16)
					if err != nil {
						return nil, err
					}
					school.TeamNumber = uint(teamNumber)
				case SCHOOL_COL_NAME:
					// Team name (track if applicable)
					trackIdx := strings.Index(trimmedCell, "(")
					teamName := strings.Trim(trimmedCell[:trackIdx], " ")
					track := ""
					if trackIdx != -1 {
						track = strings.Trim(trimmedCell[trackIdx:], " ()")
					}
					school.Name = teamName
					school.Track = track
				case TOTAL_COL_NAME:
					// Team score total
					school.TotalScore = trimmedCell
				case PLACE_COL_NAME:
					// Team placement
					school.Rank = trimmedCell
				default:
					score, err := strconv.ParseUint(trimmedCell, 10, 16)
					if err != nil {
						return nil, err
					}
					school.Scores = append(school.Scores, uint(score))
				}
			}
			parsedTable.Schools = append(parsedTable.Schools, school)
		}
		if err == io.EOF {
			break
		}
	}

	return &parsedTable, nil
}
