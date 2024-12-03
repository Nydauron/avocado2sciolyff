package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type ProgressBarValue = float64
type ProgressBarComplete = struct{}

type FileDownloadType = uint

type InputFileError struct {
	Err error
	Id  FileDownloadType
}

type FinishedDownloading struct {
	Id FileDownloadType
}

func (e InputFileError) Error() string {
	return e.Err.Error()
}

type ProgressBar struct {
	enableSpinner bool
	progress      progress.Model
	percent       float64
	spinner       spinner.Model
	label         string

	totalBytes      int64
	completedBytes  int64
	enableCheckmark bool
	err             error
}

func NewProgressBar() ProgressBar {
	s := spinner.New(spinner.WithSpinner(spinner.Dot))
	return ProgressBar{enableSpinner: true, progress: progress.New(), spinner: s, totalBytes: -1, enableCheckmark: false}
}

func (m *ProgressBar) GetSpinnerId() int {
	return m.spinner.ID()
}

func (m *ProgressBar) GetSpinnerInitTick() tea.Cmd {
	return m.spinner.Tick
}

func (m *ProgressBar) SetTotalBytes(amountBytes int64) {
	m.totalBytes = amountBytes
	if amountBytes < 0 {
		m.enableSpinner = true
	}
}

func (m *ProgressBar) SetCompletedBytes(amountBytes int64) {
	m.completedBytes = amountBytes
}

func (m *ProgressBar) SetLabel(label string) {
	m.label = label
}

func (m ProgressBar) Init() tea.Cmd {
	return nil
}

func (m ProgressBar) Update(msg tea.Msg) (ProgressBar, tea.Cmd) {
	switch msg := msg.(type) {
	case FinishedDownloading:
		m.enableCheckmark = true
		return m, nil
	case ProgressBarComplete:
		m.enableCheckmark = true
		return m, nil
	case ProgressBarValue:
		if m.totalBytes >= 0 {
			m.enableSpinner = false
		}
		m.percent = msg
		return m, nil
	case InputFileError:
		m.err = msg.Err
		return m, nil
	case spinner.TickMsg:
		if m.enableSpinner {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m ProgressBar) View() string {
	if m.err != nil {
		return fmt.Sprintf("%s \u274c %s", m.label, m.err.Error())
	}
	if m.enableSpinner {
		if m.enableCheckmark {
			return fmt.Sprintf("%s \u2714", m.label)
		}
		return fmt.Sprintf("%s %s", m.label, m.spinner.View())
	}

	scaledByteAmountTotal, byteTotalUnits := convertUpByteUnits(m.totalBytes)
	scaledByteAmountGroup, byteGroupUnits := convertUpByteUnits(m.completedBytes)
	return m.progress.ViewAs(m.percent) + fmt.Sprintf("  %4.2f%s/%4.2f%s", scaledByteAmountGroup, byteGroupUnits, scaledByteAmountTotal, byteTotalUnits)
}

const byteUnitFactor int64 = 1024

var byteUnitString = []string{"B", "kB", "MB", "GB", "TB"}

func convertUpByteUnits(amountBytes int64) (float64, string) {
	convertedByteCount := float64(amountBytes)
	factorPower := 0
	for amountBytes >= byteUnitFactor && factorPower < len(byteUnitString)-1 {
		amountBytes /= byteUnitFactor
		factorPower++
		convertedByteCount /= float64(byteUnitFactor)
	}

	return convertedByteCount, byteUnitString[factorPower]
}
