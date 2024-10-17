package parsers

import (
	"regexp"

	"github.com/Nydauron/avogado-to-sciolyff/sciolyff"
)

var numberRegex = regexp.MustCompile(`[0-9]+`)

type Table struct {
	Events  []AvogadroEvent
	Schools []sciolyff.School
}

type AvogadroEvent struct {
	Name            string
	IsMarkedAsTrial bool
}
