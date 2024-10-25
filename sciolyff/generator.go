package sciolyff

import (
	"fmt"
	"os"
	"slices"

	"github.com/Nydauron/avogado-to-sciolyff/parsers"
	"github.com/Nydauron/avogado-to-sciolyff/prompts"
	sciolyff_models "github.com/Nydauron/avogado-to-sciolyff/sciolyff/models"
)

const (
	TrackPlaceNoCalc   = 0
	TrackPlaceCalc     = 1
	TrackPlaceProvided = 2
)

func GenerateSciolyFF(table parsers.Table, groupResTable *parsers.Table) sciolyff_models.SciolyFF {
	// FIX: Assumes table and groupResTable have the same events and same teams. Should do some validation here or earlier ...
	events := make([]sciolyff_models.Event, 0)
	for _, e := range table.Events {
		isEventTrialEvent := false
		if e.IsMarkedAsTrial {
			isEventTrialEvent = prompts.EventDistingushTrialMarkerPrompt(e.Name)
		}
		events = append(events, sciolyff_models.Event{Name: e.Name, IsTrial: e.IsMarkedAsTrial && isEventTrialEvent, TrialedNormalEvent: e.IsMarkedAsTrial && !isEventTrialEvent})
	}

	var isTrackPlaceCalculationAllowed uint
	// Map of team numbers to map of scores by event name
	groupScoresByTeam := map[uint]map[string]uint{}
	if groupResTable != nil {
		isTrackPlaceCalculationAllowed = TrackPlaceProvided
		fmt.Fprintln(os.Stderr, "Table with group results was provided. Skipping track place calculation ...")
		for _, team := range groupResTable.Schools {
			// FIX: Assumes order is the same event order as overall
			scoreMap := map[string]uint{}
			for i, score := range team.Scores {
				scoreMap[groupResTable.Events[i].Name] = score
			}
			groupScoresByTeam[team.TeamNumber] = scoreMap
		}
	} else {
		if prompts.AllowCalculationTrackPlaceFromOverallPrompt() {
			isTrackPlaceCalculationAllowed = TrackPlaceCalc
		} else {
			isTrackPlaceCalculationAllowed = TrackPlaceNoCalc
		}
	}

	placings := make([]*sciolyff_models.Placing, 0)
	teamCount := uint(len(table.Schools))
	trackNames := map[string]struct{}{}
	teamCountPerTrack := map[string]uint{}
	placingsByEventByTrack := make([]map[string][]*sciolyff_models.Placing, len(events))
	for _, team := range table.Schools {
		trackNames[team.Track] = struct{}{}
		if len(events) != len(team.Scores) {
			panic(fmt.Sprintf("Score array for team \"%s\" is not the same size as number of events (%d events, %d scores)", team.Name, len(events), len(team.Scores)))
		}

		if _, ok := teamCountPerTrack[team.Track]; !ok {
			teamCountPerTrack[team.Track] = 0
		}
		teamCountPerTrack[team.Track] += 1

		for eventIdx, score := range team.Scores {
			p := sciolyff_models.Placing{Event: events[eventIdx].Name, TeamNumber: team.TeamNumber}
			p.Participated = true
			if score >= teamCount+1 { // NS
				p.Participated = false
			}
			if score >= teamCount+2 { // DQ
				p.EventDQ = true
			}
			p.Place = score
			placings = append(placings, &p)

			if placingsByEventByTrack[eventIdx] == nil {
				placingsByEventByTrack[eventIdx] = make(map[string][]*sciolyff_models.Placing)
			}
			if _, ok := placingsByEventByTrack[eventIdx][team.Track]; !ok {
				placingsByEventByTrack[eventIdx][team.Track] = []*sciolyff_models.Placing{}
			}

			placingsByEventByTrack[eventIdx][team.Track] = append(placingsByEventByTrack[eventIdx][team.Track], placings[len(placings)-1])
		}
	}
	switch isTrackPlaceCalculationAllowed {
	case TrackPlaceCalc:
		for _, eventPlacingsByTrack := range placingsByEventByTrack {
			for track, placings := range eventPlacingsByTrack {
				slices.SortFunc(placings, func(a, b *sciolyff_models.Placing) int {
					if !a.EventDQ && b.EventDQ {
						return -1
					}
					if !b.EventDQ && a.EventDQ {
						return 1
					}
					if a.EventDQ && b.EventDQ {
						return 0
					}
					if a.Participated && !b.Participated {
						return -1
					}
					if b.Participated && !a.Participated {
						return 1
					}
					if !a.Participated && !b.Participated {
						return 0
					}
					return int(a.Place) - int(b.Place)
				})

				for i, p := range placings {
					if !p.Participated {
						if p.EventDQ {
							p.TrackPlace = uint(len(placings)) + 2
						} else {
							p.TrackPlace = uint(len(placings)) + 1
						}
						continue
					}
					if p.Place == teamCount {
						p.TrackPlace = teamCountPerTrack[track]
					} else {
						p.TrackPlace = uint(i + 1)
					}
				}
			}
		}
	case TrackPlaceProvided:
		for _, eventPlacingsByTrack := range placingsByEventByTrack {
			for _, placings := range eventPlacingsByTrack {
				for _, p := range placings {
					p.TrackPlace = groupScoresByTeam[p.TeamNumber][p.Event]
				}
			}
		}
	}

	tracks := []sciolyff_models.Track{}

	for trackName := range trackNames {
		tracks = append(tracks, sciolyff_models.Track{Name: trackName})
	}

	tournament := sciolyff_models.TournamentMetadata{
		Name:      prompts.Prompt("Tournament name: "),
		ShortName: prompts.Prompt("Tournament nickname/short name: "),
		Location:  prompts.Prompt("Tournament location (host building/campus): "),
		Level:     prompts.TournamentLevelPrompt(),
		State:     prompts.StatePrompt(),
		Division:  prompts.TournamentDivisionPrompt(),
		Year:      prompts.RulesYearPrompt(),
		Date:      prompts.TournamentDatePrompt(),
	}

	copy_of_placings := make([]sciolyff_models.Placing, len(placings))
	for i, p := range placings {
		copy_of_placings[i] = *p
	}
	return sciolyff_models.SciolyFF{Tournament: tournament, Tracks: tracks, Events: events, Teams: table.Schools, Placings: copy_of_placings}
}
