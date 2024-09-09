Avocado2SciolyFF
================

A CLI tool to convert [Avogadro](https://avogadro.ws/) result tables to the [sciolyff](https://github.com/Duosmium/sciolyff) YAML format

## Install

- Ensure you have [Go 1.23](https://go.dev/doc/install) installed
- Clone this repo
- To generate a binary, run `go build`
- Alternatively, run `go run` to run the program without producing a binary

## Example Usage

Download and convert Illinois 2024 State tournament results and outputing it to file named `2024-il-state-c.yaml`:
```
avocado2sciolyff --input https://web.archive.org/web/20240421152458/https://app.avogadro.ws/il/uiuc-state-c/results/overall --output 2024-il-state-c.yaml
```

Converting Illinois 2024 State tournament results from local file and outputing it to file named `2024-il-state-c.yaml`:
```
avocado2sciolyff --input "2024 University of Illinois Urbana Champaign State (Div. C).html" --output 2024-il-state-c.yaml
```
