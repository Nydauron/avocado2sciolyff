package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"

	"github.com/Nydauron/avogado-to-sciolyff/parsers"
	"github.com/Nydauron/avogado-to-sciolyff/sciolyff"
	"github.com/Nydauron/avogado-to-sciolyff/writers"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

const (
	inputOverallFlag = "inputOverall"
	inputGroupFlag   = "inputGroup"
	outputFlag       = "output"
	csvFlag          = "csv"
	stdoutCLIName    = "-"
)

var build string
var semanticVersion = "v0.1.1" + build

func cliHandle(inputLocation string, inputByGroupLocation string, outputWriter io.Writer, isCSVFile bool) error {
	extractData := func(fileLocation string) (*parsers.Table, error) {
		var htmlBodyReader io.ReadCloser
		if u, err := url.ParseRequestURI(fileLocation); err == nil {
			fmt.Fprintln(os.Stderr, "URL detected")
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
				fmt.Fprintf(os.Stderr, "Page content recieved is not text/html UTF-8. Got instead \"%s\n", contentType)
			}
			htmlBodyReader = resp.Body
		} else if f, err := os.Open(inputLocation); err == nil {
			fmt.Fprintln(os.Stderr, "File detected")
			defer f.Close()
			htmlBodyReader = f
		} else {
			return nil, fmt.Errorf("provided input was neither a valid URL or a path to existing file: %v", inputLocation)
		}

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

	var overallResTable *parsers.Table = nil
	var groupResTable *parsers.Table = nil
	err_ch := make(chan error, 2)
	continue_ch := make(chan struct{})
	wg := sync.WaitGroup{}

	dataParser := func(err_channel chan<- error, inputPath string, table **parsers.Table) {
		t, err := extractData(inputPath)
		*table = t
		if err != nil {
			err_channel <- err
			return
		}
		wg.Done()
	}
	wg.Add(1)
	go dataParser(err_ch, inputLocation, &overallResTable)

	if inputByGroupLocation != "" {
		wg.Add(1)
		go dataParser(err_ch, inputByGroupLocation, &groupResTable)
	}
	go func() {
		defer close(continue_ch)
		wg.Wait()
	}()

	select {
	case err := <-err_ch:
		fmt.Fprintf(os.Stderr, "Error during parsing: %v", err)
		return err
	case <-continue_ch:
	}

	sciolyffDump := sciolyff.GenerateSciolyFF(*overallResTable, groupResTable)

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

func main() {
	var inputOverallLocation string
	inputByGroupLocation := ""
	outputLocation := ""
	isCSV := false
	app := &cli.App{
		Name:    "avocado2sciolyff",
		Usage:   "A tool to turn table results on Avogadro to sciolyff results",
		Version: semanticVersion,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        csvFlag,
				Usage:       "File passed in is a CSV rather than an HTML file",
				Destination: &isCSV,
			},
			&cli.StringFlag{
				Name:        inputOverallFlag,
				Aliases:     []string{"iO"},
				Usage:       "The URL or path to the HTML file containing the table of overall results to convert",
				Destination: &inputOverallLocation,
				Required:    true,
			},
			&cli.StringFlag{
				Name:        inputGroupFlag,
				Aliases:     []string{"iG"},
				Usage:       "The URL or path to the HTML file containing the table of results by grouping/track to convert",
				Destination: &inputByGroupLocation,
			},
			&cli.StringFlag{
				Name:        outputFlag,
				Aliases:     []string{"o"},
				Usage:       "The location to write the YAML result. Can be a file path or \"-\" (for stdout).",
				Required:    true,
				Destination: &outputLocation,
			},
		},
		Action: func(cCtx *cli.Context) error {
			if outputLocation == "" {
				return fmt.Errorf("output not set")
			}
			var outputWriter io.WriteCloser = os.Stdout
			if outputLocation != stdoutCLIName {
				outputWriter = writers.NewLazyWriteCloser(func() (io.WriteCloser, error) {
					return os.OpenFile(outputLocation, os.O_CREATE|os.O_WRONLY, 0644)
				})
			}
			return cliHandle(inputOverallLocation, inputByGroupLocation, outputWriter, isCSV)
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
