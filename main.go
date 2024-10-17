package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/Nydauron/avogado-to-sciolyff/parsers"
	"github.com/Nydauron/avogado-to-sciolyff/sciolyff"
	"github.com/Nydauron/avogado-to-sciolyff/writers"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

const (
	inputFlag     = "input"
	outputFlag    = "output"
	csvFlag       = "csv"
	stdoutCLIName = "-"
)

var build string
var semanticVersion = "v0.1.0-dev" + build

func cliHandle(inputLocation string, outputWriter io.Writer, isCSVFile bool) error {
	var htmlBodyReader io.ReadCloser
	if u, err := url.ParseRequestURI(inputLocation); err == nil {
		fmt.Fprintln(os.Stderr, "URL detected")
		rawURL := u.String()
		resp, err := http.Get(rawURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error occurred when trying to fetch page: %v\n", err)
			os.Exit(2)
			return nil
		}

		if resp.StatusCode >= 400 {
			return fmt.Errorf("invalid HTTP status code received: %v", resp.Status)
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
		return fmt.Errorf("provided input was neither a valid URL or a path to existing file: %v", inputLocation)
	}

	var table *parsers.Table
	if isCSVFile {
		var err error
		table, err = parsers.ParseCSV(htmlBodyReader)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Cell did not contain number: %v\n", err)
			os.Exit(4)
			return nil
		}
	} else {
		var err error
		table, err = parsers.ParseHTML(htmlBodyReader)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Cell did not contain number: %v\n", err)
			os.Exit(4)
			return nil
		}
	}

	sciolyffDump := sciolyff.GenerateSciolyFF(*table)

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
	var inputLocation string
	var outputLocation = ""
	var isCSV = false
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
				Name:        inputFlag,
				Aliases:     []string{"i"},
				Usage:       "The URL or path to the HTML file containing the table to convert",
				Destination: &inputLocation,
				Required:    true,
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
			return cliHandle(inputLocation, outputWriter, isCSV)
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
