package parsers

import (
	"regexp"

	sciolyff_models "github.com/Nydauron/avocado2sciolyff/sciolyff/models"
)

var numberRegex = regexp.MustCompile(`[0-9]+`)

type Table struct {
	Events  []AvogadroEvent
	Schools []sciolyff_models.School
}

type AvogadroEvent struct {
	Name            string
	IsMarkedAsTrial bool
}
