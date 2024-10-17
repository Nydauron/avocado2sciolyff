package prompts

import (
	"bufio"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	eventIsTrial    = true
	eventWasTrialed = false
)

func EventDistingushTrialMarkerPrompt(eventName string) bool {
	for {
		userInput := Prompt(fmt.Sprintf("Event %s had a trial marker. Was this event a trial event (1) or was the event trialed (2)? ", eventName))
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

func TournamentDatePrompt() string {
	for {
		userInput := Prompt("Tournament date: ")
		_, err := time.Parse(time.DateOnly, userInput)
		if err == nil {
			return userInput
		}
	}
}

func RulesYearPrompt() int {
	for {
		userInput := Prompt("Rules Year: ")
		parsedYear, err := strconv.Atoi(userInput)
		if err == nil {
			return parsedYear
		}
	}
}

func TournamentDivisionPrompt() string {
	translatedDivision := ""
	for translatedDivision == "" {
		userInput := strings.ToUpper(Prompt("Tournament division (a, b, c): "))
		if userInput[0] >= 'A' || userInput[0] <= 'C' {
			translatedDivision = userInput[:1]
		}
	}
	return translatedDivision
}

func TournamentLevelPrompt() string {
	translatedLevel := ""
	for translatedLevel == "" {
		userInput := Prompt("Tournament level (i, r, s, n): ")
		translatedLevel = TranslateLevelAbbrevToFull(strings.ToLower(userInput)[0])
	}
	return translatedLevel
}

func StatePrompt() string {
	translatedState := ""
	for translatedState == "" {
		userInput := strings.ToUpper(Prompt("State: "))
		if slices.Contains(stateAbbreviations, userInput) {
			translatedState = userInput
		} else if slices.Contains(stateNames, userInput) {
			translatedState = stateMapping[userInput]
		}
	}
	return translatedState
}

func TranslateLevelAbbrevToFull(a byte) string {
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

func AllowCalculationTrackPlaceFromOverallPrompt() bool {
	for {
		userInput := Prompt("Calculate track placements based on overall score? (y/N) ")
		userInput = strings.ToLower(userInput)
		if userInput == "y" {
			return true
		}
		if userInput == "n" || userInput == "" {
			return false
		}
	}
}

func Prompt(message string) string {
	fmt.Fprint(os.Stderr, message)
	buf := bufio.NewReader(os.Stdin)
	input, _ := buf.ReadString('\n')
	return strings.TrimRight(input, "\n")
}
