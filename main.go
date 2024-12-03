package main

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	// "log"
	"net/http"
	"net/url"
	"os"
	"slices"
	"sync"

	"github.com/Nydauron/avocado2sciolyff/parsers"
	"github.com/Nydauron/avocado2sciolyff/prompts"
	"github.com/Nydauron/avocado2sciolyff/sciolyff"
	sciolyff_models "github.com/Nydauron/avocado2sciolyff/sciolyff/models"
	"github.com/Nydauron/avocado2sciolyff/ui"
	"github.com/Nydauron/avocado2sciolyff/writers"

	// "github.com/Nydauron/avocado2sciolyff/writers"
	// "github.com/urfave/cli/v2"
	// "golang.org/x/text/cases"
	"gopkg.in/yaml.v3"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	inputOverallFlag = "inputOverall"
	inputGroupFlag   = "inputGroup"
	outputFlag       = "output"
	csvFlag          = "csv"
	stdoutCLIName    = "-"
)

var build string
var semanticVersion = "v0.2.0-dev" + build

var p *tea.Program

type FileDownloadType = uint

type ProgressBarUpdate struct {
	id                  FileDownloadType
	updatedPercent      ui.ProgressBarValue
	totalByteCount      int64
	downloadedByteCount int64
}

type ProgressReader struct {
	progressBarId FileDownloadType
	total         int64
	downloaded    int64
	reader        io.ReadCloser
}

func (r *ProgressReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.downloaded += int64(n)
	r.onUpdate()
	return n, err
}

func (r *ProgressReader) Close() error {
	return r.Close()
}

func (r *ProgressReader) onUpdate() {
	if p != nil {
		p.Send(ProgressBarUpdate{
			id:                  r.progressBarId,
			updatedPercent:      (float64(r.downloaded) / float64(r.total)),
			totalByteCount:      r.total,
			downloadedByteCount: r.downloaded,
		})
	}
}

func fileFetch(fileLocation string, progressBarId *FileDownloadType) (io.ReadCloser, error) {
	var htmlBodyReader io.ReadCloser
	fileSize := int64(-1)
	if u, err := url.ParseRequestURI(fileLocation); err == nil {
		if !slices.Contains([]string{"http", "https"}, u.Scheme) {
			return nil, fmt.Errorf("URL is not of HTTP schema (got %q instead)", u.Scheme)
		}
		rawURL := u.String()
		resp, err := http.Get(rawURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error occurred when trying to fetch page: %v\n", err)
			return nil, err
		}

		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("invalid HTTP status code received: %v", resp.Status)
		}
		defer resp.Body.Close()
		contentType := resp.Header.Get("content-type")
		expectedContent := "text/html; charset=UTF-8"
		if contentType != expectedContent {
			fmt.Fprintf(os.Stderr, "Page content recieved is not text/html UTF-8. Got instead %q\n", contentType)
		}
		// resp.ContentLength is currently always being set to -1. No clue why
		fileSize = resp.ContentLength
		htmlBodyReader = resp.Body
	} else if f, err := os.Open(fileLocation); err == nil {
		fmt.Fprintln(os.Stderr, "File detected")
		defer f.Close()
		stats, err := f.Stat()
		if err == nil {
			fileSize = stats.Size()
		}
		htmlBodyReader = f
	} else {
		return nil, fmt.Errorf("provided input was neither a valid URL or a path to existing file: %v", fileLocation)
	}

	if progressBarId != nil {
		htmlBodyReader = &ProgressReader{
			progressBarId: *progressBarId,
			total:         int64(fileSize),
			downloaded:    0,
			reader:        htmlBodyReader,
		}
	}
	return htmlBodyReader, nil
}

func extractDataHandler(isCSVFile bool) func(string, *FileDownloadType) (*parsers.Table, error) {
	return func(fileLocation string, progressBarId *FileDownloadType) (*parsers.Table, error) {
		var table *parsers.Table
		if isCSVFile {
			var err error
			table, err = parsers.ParseCSV(htmlBodyReader)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Cell did not contain number: %v\n", err)
				os.Exit(4)
			}
		} else {
			var err error
			table, err = parsers.ParseHTML(htmlBodyReader)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Cell did not contain number: %v\n", err)
				os.Exit(4)
			}
		}

		return table, nil
	}
}

// func cliHandle(inputLocation string, inputByGroupLocation string, outputWriter io.Writer, isCSVFile bool) error {
// 	// use p.Send to send update messages to UI
// 	extractData := extractDataHandler(isCSVFile)
//
// 	var overallResTable *parsers.Table = nil
// 	var groupResTable *parsers.Table = nil
// 	err_ch := make(chan error, 2)
// 	continue_ch := make(chan struct{})
// 	wg := sync.WaitGroup{}
//
// 	dataParser := func(err_channel chan<- error, inputPath string, table **parsers.Table) {
// 		t, err := extractData(inputPath, nil)
// 		*table = t
// 		if err != nil {
// 			err_channel <- err
// 			return
// 		}
// 		wg.Done()
// 	}
// 	wg.Add(1)
// 	go dataParser(err_ch, inputLocation, &overallResTable)
//
// 	if inputByGroupLocation != "" {
// 		wg.Add(1)
// 		go dataParser(err_ch, inputByGroupLocation, &groupResTable)
// 	}
// 	go func() {
// 		defer close(continue_ch)
// 		wg.Wait()
// 	}()
//
// 	select {
// 	case err := <-err_ch:
// 		fmt.Fprintf(os.Stderr, "Error during parsing: %v", err)
// 		return err
// 	case <-continue_ch:
// 	}
//
// 	sciolyffDump := sciolyff.GenerateSciolyFF(*overallResTable, groupResTable)
//
// 	outputWriter.Write([]byte("###\n# This YAML file was auto-generated by avocado2sciolyff " + semanticVersion + "\n###\n"))
// 	yamlEncoder := yaml.NewEncoder(outputWriter)
// 	yamlEncoder.SetIndent(2)
// 	err := yamlEncoder.Encode(&sciolyffDump)
// 	if err != nil {
// 		fmt.Fprintf(os.Stderr, "Encoding to YAML failed: %v", err)
// 		os.Exit(3)
// 		return nil
// 	}
//
// 	err = yamlEncoder.Close()
// 	if err != nil {
// 		fmt.Fprintf(os.Stderr, "Encoding to YAML failed on close: %v", err)
// 		os.Exit(3)
// 		return nil
// 	}
//
// 	return nil
// }

func main() {
	f, err := tea.LogToFile("debug.log", "debug")
	if err != nil {
		fmt.Println("fatal:", err)
		os.Exit(1)
	}
	defer f.Close()
	p = tea.NewProgram(NewOriginModel(ArgumentInputs{inputOverallLocation: "https://app.avogadro.ws/il/phs-invitational-c/results/overall", inputGroupLocation: "https://app.avogadro.ws/il/phs-invitational-c/results/groups"}))

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Bubble tea error: %v\n", err)
		os.Exit(1)
	}
	// var inputOverallLocation string
	// inputByGroupLocation := ""
	// outputLocation := ""
	// isCSV := false
	// app := &cli.App{
	// 	Name:    "avocado2sciolyff",
	// 	Usage:   "A tool to turn table results on Avogadro to sciolyff results",
	// 	Version: semanticVersion,
	// 	Flags: []cli.Flag{
	// 		&cli.BoolFlag{
	// 			Name:        csvFlag,
	// 			Usage:       "File passed in is a CSV rather than an HTML file",
	// 			Destination: &isCSV,
	// 		},
	// 		&cli.StringFlag{
	// 			Name:        inputOverallFlag,
	// 			Aliases:     []string{"iO"},
	// 			Usage:       "The URL or path to the HTML file containing the table of overall results to convert",
	// 			Destination: &inputOverallLocation,
	// 			Required:    true,
	// 		},
	// 		&cli.StringFlag{
	// 			Name:        inputGroupFlag,
	// 			Aliases:     []string{"iG"},
	// 			Usage:       "The URL or path to the HTML file containing the table of results by grouping/track to convert",
	// 			Destination: &inputByGroupLocation,
	// 		},
	// 		&cli.StringFlag{
	// 			Name:        outputFlag,
	// 			Aliases:     []string{"o"},
	// 			Usage:       "The location to write the YAML result. Can be a file path or \"-\" (for stdout).",
	// 			Required:    true,
	// 			Destination: &outputLocation,
	// 		},
	// 	},
	// 	Action: func(cCtx *cli.Context) error {
	// 		if outputLocation == "" {
	// 			return fmt.Errorf("output not set")
	// 		}
	// 		var outputWriter io.WriteCloser = os.Stdout
	// 		if outputLocation != stdoutCLIName {
	// 			outputWriter = writers.NewLazyWriteCloser(func() (io.WriteCloser, error) {
	// 				return os.OpenFile(outputLocation, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	// 			})
	// 		}
	// 		return cliHandle(inputOverallLocation, inputByGroupLocation, outputWriter, isCSV)
	// 	},
	// }
	//
	// if err := app.Run(os.Args); err != nil {
	// 	log.Fatal(err)
	// }
}

type ModelState uint

const (
	RetrievingFilesState ModelState = 1
	PrePromptState       ModelState = 2
	PromptState          ModelState = 3
	WriteToFileState     ModelState = 4
)

type originModel struct {
	state               ModelState
	promptIdx           int
	prompts             []ui.Prompt
	fileDownloadOverall ui.ProgressBar
	fileDownloadGroup   ui.ProgressBar

	inputs ArgumentInputs

	overallTable *parsers.Table
	groupTable   *parsers.Table

	validatedPromptData *sciolyff_models.PromptData
}

type ArgumentInputs struct {
	inputOverallLocation string
	inputGroupLocation   string
	isCSVFile            bool

	outputFileLocation string
}

const OverallInputType FileDownloadType = 1
const GroupInputType FileDownloadType = 2

type FileDownloadProgress struct {
	inputType FileDownloadType
	progress  float64
}

type SetPrompts struct {
	prompts []ui.InputData
}

func NewOriginModel(inputs ArgumentInputs) originModel {
	fileDownloadGroup := ui.NewProgressBar()
	fileDownloadOverall := ui.NewProgressBar()

	fileDownloadGroup.SetLabel(inputs.inputGroupLocation)
	fileDownloadOverall.SetLabel(inputs.inputOverallLocation)

	return originModel{
		state: RetrievingFilesState, promptIdx: 0,
		fileDownloadGroup:   fileDownloadGroup,
		fileDownloadOverall: fileDownloadOverall,
		inputs:              inputs,
		validatedPromptData: &sciolyff_models.PromptData{},
	}
}

func (m originModel) Init() tea.Cmd {
	return tea.Batch(func() tea.Msg {
		extractData := extractDataHandler(m.inputs.isCSVFile)

		var overallResTable *parsers.Table = nil
		var groupResTable *parsers.Table = nil
		err_ch := make(chan error, 2)
		continue_ch := make(chan struct{})
		wg := sync.WaitGroup{}

		dataParser := func(err_channel chan<- error, inputPath string, table **parsers.Table, id FileDownloadType) {
			t, err := extractData(inputPath, &id)
			*table = t
			if err != nil {
				err_channel <- err
				return
			}
			wg.Done()
		}
		wg.Add(1)
		go dataParser(err_ch, m.inputs.inputOverallLocation, &overallResTable, OverallInputType)

		if m.inputs.inputGroupLocation != "" {
			wg.Add(1)
			go dataParser(err_ch, m.inputs.inputGroupLocation, &groupResTable, GroupInputType)
		}
		go func() {
			defer close(continue_ch)
			wg.Wait()
		}()

		select {
		case err := <-err_ch:
			fmt.Fprintf(os.Stderr, "Error during parsing: %v", err)
			return fmt.Errorf("")
			return err
		case <-continue_ch:
		}

		return FinishDownloading{
			overallTable: overallResTable,
			groupTable:   groupResTable,
		}
	}, m.fileDownloadGroup.GetSpinnerInitTick(), m.fileDownloadOverall.GetSpinnerInitTick())
}

type FinishDownloading struct {
	overallTable *parsers.Table
	groupTable   *parsers.Table
}

func (m originModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd = nil
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "enter":
			if m.state == PromptState {
				current := m.prompts[m.promptIdx]

				if current.IsValueValid() {
					nextState := m.Next()
					m.state = nextState
					if nextState == WriteToFileState {
						return m, func() tea.Msg {
							var outputWriter io.WriteCloser = os.Stdout
							if m.inputs.outputFileLocation != stdoutCLIName {
								outputWriter = writers.NewLazyWriteCloser(func() (io.WriteCloser, error) {
									return os.OpenFile(m.inputs.outputFileLocation, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
								})
							}
							sciolyffDump := sciolyff.GenerateSciolyFF(*m.overallTable, m.groupTable, *m.validatedPromptData)

							outputWriter.Write([]byte(fmt.Sprintf("###\n# This YAML file was auto-generated by avocado2sciolyff " + semanticVersion + "\n###\n")))
							yamlEncoder := yaml.NewEncoder(outputWriter)
							yamlEncoder.SetIndent(2)
							err := yamlEncoder.Encode(&sciolyffDump)
							if err != nil {
								fmt.Fprintf(os.Stderr, "Encoding to YAML failed: %v", err)
								os.Exit(3)
								return nil
							}

							err = yamlEncoder.Close()
							if err != nil {
								fmt.Fprintf(os.Stderr, "Encoding to YAML failed on close: %v", err)
								os.Exit(3)
								return nil
							}
							return nil
						}
					}

				}
			}
			return m, nil
		}
	case FileDownloadProgress:
		switch msg.inputType {
		case OverallInputType:
			m.fileDownloadOverall, cmd = m.fileDownloadOverall.Update(msg)
		case GroupInputType:
			m.fileDownloadGroup, cmd = m.fileDownloadGroup.Update(msg)
		}
		return m, cmd
	case ProgressBarUpdate:
		switch msg.id {
		case OverallInputType:
			m.fileDownloadOverall, cmd = m.fileDownloadOverall.Update(msg.updatedPercent)
			return m, cmd
		case GroupInputType:
			m.fileDownloadGroup, cmd = m.fileDownloadGroup.Update(msg.updatedPercent)
			return m, cmd
		}
	case FinishDownloading:
		m.state = PrePromptState
		m.overallTable = msg.overallTable
		m.groupTable = msg.groupTable
		m.fileDownloadOverall, _ = m.fileDownloadOverall.Update(ui.ProgressBarComplete{})
		m.fileDownloadGroup, _ = m.fileDownloadGroup.Update(ui.ProgressBarComplete{})
		return m, GeneratePrompts(m.validatedPromptData, m.overallTable.Info)
	case spinner.TickMsg:
		if m.state == RetrievingFilesState {
			switch msg.ID {
			case m.fileDownloadOverall.GetSpinnerId():
				m.fileDownloadOverall, cmd = m.fileDownloadOverall.Update(msg)
			case m.fileDownloadGroup.GetSpinnerId():
				m.fileDownloadGroup, cmd = m.fileDownloadGroup.Update(msg)
			}
			return m, cmd
		}
	case SetPrompts:
		if m.state == PrePromptState {
			m.prompts = nil
			for _, promptData := range msg.prompts {
				m.prompts = append(m.prompts, ui.NewPrompt(promptData))
			}
			m.state = PromptState
		}
	}

	if m.state == PromptState {
		current := &m.prompts[m.promptIdx]
		current.Input, cmd = current.Input.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *originModel) Next() ModelState {
	if m.promptIdx >= len(m.prompts)-1 {
		m.promptIdx = 0
		return WriteToFileState
	}
	m.promptIdx++
	return PromptState
}

func (m originModel) View() string {
	s := ""
	s += fmt.Sprintf("%s\n%s\n", m.fileDownloadOverall.View(), m.fileDownloadGroup.View())
	if m.state == RetrievingFilesState || m.state == PrePromptState {
		return s
	}
	for i, p := range m.prompts {
		if m.state == PromptState && i == m.promptIdx {
			err_str := "\u2714"
			if err := p.ParseValue(); err != nil {
				err_str = fmt.Sprintf("\u274c %s", err.Error())
			}
			s += fmt.Sprintf("%s %s\n", p.View(), err_str)
			break
		}
		s += fmt.Sprintf("%s: %s\n", p.Data.Question, p.GetValue())
	}
	if m.state == WriteToFileState {
		s += "Congrats! Here's a spork\n"
	}
	return s
}

func GeneratePrompts(dataLocation *sciolyff_models.PromptData, initialValues parsers.AvogadroTournamentInfo) tea.Cmd {
	return func() tea.Msg {
		return SetPrompts{
			prompts: []ui.InputData{
				{
					Question:     "Name",
					DefaultValue: initialValues.Name,
					Parse: func(s string) error {
						if len(s) == 0 {
							return fmt.Errorf("required")
						}
						dataLocation.Name = s
						return nil
					},
				},
				{
					Question: "Tournament nickname/short name",
					Parse: func(s string) error {
						if len(s) == 0 {
							return fmt.Errorf("required")
						}
						dataLocation.ShortName = s
						return nil
					},
				},
				{
					Question: "Location",
					Parse: func(s string) error {
						dataLocation.Location = s
						return nil
					},
				},
				{
					Question: "Tournament level (i, r, s, n)",
					Parse: func(s string) error {
						if len(s) == 0 {
							return fmt.Errorf("required")
						}
						c := strings.ToLower(s)[0]
						inviteType := prompts.TranslateLevelAbbrevToFull(c)
						if inviteType == "" {
							return fmt.Errorf("value must be i, r, s, or n")
						}
						dataLocation.Level = inviteType
						return nil
					},
				},
				{
					Question:     "State",
					DefaultValue: initialValues.State,
					Parse: func(s string) error {
						translatedState := ""
						state_str := strings.ToUpper(s)
						if slices.Contains(prompts.StateAbbreviations, state_str) {
							translatedState = state_str
						} else if slices.Contains(prompts.StateNames, state_str) {
							translatedState = prompts.StateMapping[state_str]
						}
						if translatedState == "" {
							return fmt.Errorf("unknown state")
						}
						dataLocation.State = translatedState
						return nil
					},
				},
				{
					Question:     "Tournament division (a, b, c)",
					DefaultValue: initialValues.Division,
					Parse: func(s string) error {
						if len(s) == 0 {
							return fmt.Errorf("required")
						}
						c := strings.ToLower(s)[0]
						if c < 'a' || c > 'c' {
							return fmt.Errorf("unknown tournament division")
						}

						dataLocation.Division = strings.ToUpper(s[:1])
						return nil
					},
				},
				{
					Question: "Rules year",
					Parse: func(s string) error {
						year, err := strconv.Atoi(s)
						if err != nil {
							return fmt.Errorf("Parsing error")
						}
						dataLocation.Year = year
						return nil
					},
				},
				{
					Question: "Tournament date",
					Parse: func(s string) error {
						_, err := time.Parse(time.DateOnly, s)
						if err != nil {
							return fmt.Errorf("Not a valid YYYY-mm-dd date")
						}

						dataLocation.Date = s
						return nil
					},
				},
			},
		}
	}
}
