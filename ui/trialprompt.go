package ui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type UpdateSelection struct {
	Idx int
}

type Selection struct {
	selections []SelectionOption

	selected      int
	displayInline bool
}

type SelectionOption struct {
	key         string
	displayText string
}

func (m *Selection) GetKey() *string {
	if m.selected < 0 || m.selected >= len(m.selections) {
		return nil
	}
	return &m.selections[m.selected].key
}

func (m Selection) Init() tea.Cmd {
	return nil
}

func (m Selection) Update(msg tea.Msg) (Selection, tea.Cmd) {
	switch msg := msg.(type) {
	case UpdateSelection:
		m.selected = msg.Idx
		return m, nil
	}
	return m, nil
}

func (m Selection) View() string {
	s := ""
	selectionStrArr := make([]string, len(m.selections))
	for i, selection := range m.selections {
		marker := " "
		if i == m.selected {
			marker = "x"
		}
		selectionStrArr[i] = fmt.Sprintf("[%s] %s", marker, selection.displayText)
	}
	if m.displayInline {
		s += strings.Join(selectionStrArr, " ")
	} else {
		s += strings.Join(selectionStrArr, "\n")
	}

	return s

}

type TrialPrompt struct {
	EventName string

	// if false, assumes to be a trial event
	IsTrialed bool

	input  Selection
	parser func(bool) error
}

const TrialEventKey = "trial"
const EventTrialedKey = "trialed"

func NewTrialPrompt(eventName string, parser func(bool) error) TrialPrompt {
	return TrialPrompt{
		EventName: eventName,
		input: Selection{
			selections: []SelectionOption{
				{
					key:         TrialEventKey,
					displayText: "Trial event",
				},
				{
					key:         EventTrialedKey,
					displayText: "Event was trialed",
				},
			},
			selected:      0,
			displayInline: true,
		},
		parser: parser,
	}
}

func (m *TrialPrompt) ParseValue() error {
	return m.parser(m.IsTrialed)
}

func (m TrialPrompt) Init() tea.Cmd {
	return nil
}

func (m TrialPrompt) Update(msg tea.Msg) (TrialPrompt, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "1":
			fallthrough
		case "2":
			idx, _ := strconv.Atoi(msg.String())
			var cmd tea.Cmd = nil
			m.input, cmd = m.input.Update(UpdateSelection{Idx: idx - 1})
			key := m.input.GetKey()
			if key != nil && *key == EventTrialedKey {
				m.IsTrialed = true
			} else {
				m.IsTrialed = false
			}
			m.ParseValue()
			return m, cmd

		}
	}
	return m, nil
}

func (m TrialPrompt) View() string {
	return fmt.Sprintf("%s %s", m.EventName, m.input.View())
}
