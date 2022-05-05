package main

import (
	"encoding/json"
	"fmt"
	"github.com/olekukonko/tablewriter"
	"golang.org/x/term"
	"os"
	"runtime"
	"strconv"
	"strings"
)

// printToCli will print the outcomes to CLI in the selected format
func printToCli(outcomes []Outcome, format string) {
	switch strings.ToLower(format) {
	case "console":
		for index, outcome := range outcomes {
			tablePrintOutcomeToCLI(outcome)
			if index < len(outcomes)-1 {
				width, _, _ := term.GetSize(0)
				for i := 0; i < width-14; i++ {
					fmt.Print("*")
				}
				fmt.Print("\n")
			}
		}
	case "json":
		if len(outcomes) == 1 {
			prettyPrintJsonToCLI(outcomes[0])
		} else {
			prettyPrintJsonToCLI(outcomes)
		}
	case "har":
		prettyPrintHarToCLI(outcomes)
	}

}

// prettyPrintJsonToCLI will print the probe outcome in JSON to the CLI
func prettyPrintJsonToCLI(outcomes interface{}) {
	data, err := json.MarshalIndent(outcomes, "", "\t")
	if err != nil {
		fmt.Println("Could not marshal the output: ", err.Error())
		os.Exit(1)
	}
	fmt.Println(string(data))
}

func prettyPrintHarToCLI(outcomes []Outcome) {
	prettyPrintJsonToCLI(toHar(outcomes))
}

func buildTable(header ...string) *tablewriter.Table {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	width, _, _ := term.GetSize(0)
	table.SetColWidth(width/2 - 10)
	table.SetColMinWidth(0, (width)/2-10)
	table.SetColMinWidth(1, width/2-10)
	table.SetAutoWrapText(true)
	table.SetBorders(tablewriter.Border{Left: true, Right: true, Top: true})
	table.SetColumnAlignment([]int{tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_LEFT})
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAutoWrapText(true)
	table.SetHeader(header)
	if runtime.GOOS != "windows" {
		table.SetColumnColor(tablewriter.Color(tablewriter.Normal, tablewriter.FgCyanColor),
			tablewriter.Color(tablewriter.Normal))
		table.SetHeaderColor(tablewriter.Color(tablewriter.Bold, tablewriter.BgHiBlackColor, tablewriter.FgHiWhiteColor),
			tablewriter.Color(tablewriter.Bold, tablewriter.BgHiBlackColor, tablewriter.FgHiWhiteColor))
	}
	return table
}

func appendError(table *tablewriter.Table, label string, val string) {
	if runtime.GOOS == "windows" {
		table.Append([]string{label, val})
	} else {
		table.Rich([]string{label, val}, []tablewriter.Colors{
			{tablewriter.Normal, tablewriter.FgHiRedColor},
			{}})
	}

}
func appendSuccess(table *tablewriter.Table, label string, val string) {
	if runtime.GOOS == "windows" {
		table.Append([]string{label, val})
	} else {
		table.Rich([]string{label, val}, []tablewriter.Colors{
			{tablewriter.Normal, tablewriter.FgHiGreenColor},
			{}})
	}

}

func tablePrintOutcomeToCLI(outcome Outcome) {
	table := buildTable("Request", "Values")
	table.Append([]string{"Method", outcome.Requester.Method})
	table.Append([]string{"URL", outcome.Requester.Url})
	table.Append([]string{"Timeout", outcome.Requester.Timeout.String()})
	table.Render()
	table = buildTable("ResponseOutcomeWrapper", "Values")
	table.Append([]string{"IP Address", outcome.IpAddress})
	table.Append([]string{"Status", strconv.Itoa(outcome.StatusCode)})
	table.Append([]string{"Size", byteCountDecimal(outcome.Size)})
	if outcome.Err != nil {
		appendError(table, "Error", outcome.Err.Error())
	}
	table.Render()
	table = buildTable("Metrics", "Values")
	if len(outcome.Annotations) == 0 && len(outcome.Checks) == 0 {
		table.SetBorders(tablewriter.Border{Left: true, Right: true, Top: true, Bottom: true})
	}
	table.Append([]string{"DNS", outcome.Metrics.DNS.String()})
	table.Append([]string{"Conn", outcome.Metrics.Conn.String()})
	table.Append([]string{"TLS", outcome.Metrics.TLS.String()})
	table.Append([]string{"TTFB", outcome.Metrics.TTFB.String()})
	table.Append([]string{"Transfer", outcome.Metrics.Transfer.String()})
	table.Append([]string{"RT", outcome.Metrics.RT.String()})
	table.Render()
	if len(outcome.Annotations) > 0 {
		table = buildTable("Annotations", "Values")
		for _, annotation := range outcome.Annotations {
			table.Append([]string{annotation.Annotation, fmt.Sprintln(annotation.Text)})
		}
		if len(outcome.Checks) == 0 {
			table.SetBorders(tablewriter.Border{Left: true, Right: true, Top: true, Bottom: true})
		}
		table.Render()
	}

	if len(outcome.Checks) > 0 {
		table = buildTable("Assertions", "Results")
		for _, check := range outcome.Checks {
			output := fmt.Sprint(check.Output)
			if check.Success {
				appendSuccess(table, check.Assertion, output)
			} else {
				appendError(table, check.Assertion, output)
			}

		}
		table.SetBorders(tablewriter.Border{Left: true, Right: true, Top: true, Bottom: true})
		table.Render()
	}
}

// byteCountDecimal will make the payload size human-readable
func byteCountDecimal(b int) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := unit, 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "kMGTPE"[exp])
}
