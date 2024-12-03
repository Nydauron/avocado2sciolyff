package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type Input interface {
	Init() tea.Cmd
	Focus() tea.Cmd
	SetValue(string)
	GetValue() string
	Update(tea.Msg) (Input, tea.Cmd)
	View() string
}

type InputData struct {
	Question     string
	DefaultValue string

	// Function that gets run on the input value
	Parse func(string) error
}

type Prompt struct {
	Input textinput.Model
	Data  InputData
}

func NewPrompt(inputData InputData) Prompt {
	input := textinput.New()
	input.Prompt = fmt.Sprintf("%s: ", inputData.Question)
	input.Placeholder = inputData.DefaultValue
	input.Focus()
	return Prompt{Data: inputData, Input: input}
}

func (m Prompt) Focus() tea.Cmd {
	return m.Input.Focus()
}

func (m Prompt) SetValue(value string) {
	m.Input.SetValue(value)
}

func (m Prompt) GetValue() string {
	if m.Input.Value() != "" {
		return m.Input.Value()
	}
	return m.Data.DefaultValue
}

func (m *Prompt) IsValueValid() bool {
	return m.ParseValue() == nil
}

func (m *Prompt) ParseValue() error {
	val := m.Input.Value()
	if val == "" {
		val = m.Data.DefaultValue
	}
	return m.Data.Parse(val)
}

func (m Prompt) Init() tea.Cmd {
	return nil
}

func (m Prompt) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.Input, cmd = m.Input.Update(msg)
	return m, cmd
}

func (m Prompt) View() string {
	return m.Input.View()
}
