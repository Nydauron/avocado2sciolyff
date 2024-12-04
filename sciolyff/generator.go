package sciolyff

import (
	"fmt"
	"slices"

	"github.com/Nydauron/avocado2sciolyff/parsers"
	sciolyff_models "github.com/Nydauron/avocado2sciolyff/sciolyff/models"
)

const (
	TrackPlaceNoCalc   = 0
	TrackPlaceCalc     = 1
	TrackPlaceProvided = 2
)

func GenerateSciolyFF(table parsers.Table, groupResTable *parsers.Table, promptData sciolyff_models.PromptData) sciolyff_models.SciolyFF {
	// FIX: Assumes table and groupResTable have the same events and same teams. Should do some validation here or earlier ...
	events := make([]sciolyff_models.Event, 0)
	for _, e := range table.Events {
		isEventTrialEvent := false
		if e.IsMarkedAsTrial {
			if eventTrialed, ok := promptData.TrialEventsTrialed[e.Name]; ok {
				isEventTrialEvent = !eventTrialed
			}
		}
		events = append(events, sciolyff_models.Event{Name: e.Name, IsTrial: e.IsMarkedAsTrial && isEventTrialEvent, TrialedNormalEvent: e.IsMarkedAsTrial && !isEventTrialEvent})
	}

	var isTrackPlaceCalculationAllowed uint
	// Map of team numbers to map of scores by event name
	groupScoresByTeam := map[uint]map[string]uint{}
	if groupResTable != nil {
		isTrackPlaceCalculationAllowed = TrackPlaceProvided
		for _, team := range groupResTable.Schools {
			// FIX: Assumes order is the same event order as overall
			scoreMap := map[string]uint{}
			for i, score := range team.Scores {
				scoreMap[groupResTable.Events[i].Name] = score
			}
			groupScoresByTeam[team.TeamNumber] = scoreMap
		}
	} else {
		if promptData.CalculateGroupsFromOverall {
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
			// If a team gets awarded P points Participated must be true and Points must not be set
			// NOTE: If a single team got last, it is impossible to know whether the team got last or P points
			if score < teamCount {
				p.Place = score
			}
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
					if a.Place == 0 && b.Place != 0 {
						return 1
					}
					if b.Place == 0 && a.Place != 0 {
						return -1
					}
					return int(a.Place) - int(b.Place)
				})

				for i, p := range placings {
					if !p.Participated {
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
		Name:      promptData.Name,
		ShortName: promptData.ShortName,
		Location:  promptData.Location,
		Level:     promptData.Level,
		State:     promptData.State,
		Division:  promptData.Division,
		Year:      promptData.Year,
		Date:      promptData.Date,
	}

	copy_of_placings := make([]sciolyff_models.Placing, len(placings))
	for i, p := range placings {
		copy_of_placings[i] = *p
	}
	return sciolyff_models.SciolyFF{Tournament: tournament, Tracks: tracks, Events: events, Teams: table.Schools, Placings: copy_of_placings}
}
