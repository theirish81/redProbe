package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

var (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorGreen  = "\033[32m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[37m"
)

func printToCli(outcome Outcome) {
	switch outcome.Requester.Format {
	case "json":
		prettyPrintJsonToCLI(outcome)
	default:
		prettyPrintOutcomeToCLI(outcome)
	}
	if !outcome.isSuccess() {
		os.Exit(1)
	}
}

// prettyPrintJsonToCLI will print the probe outcome in JSON to the CLI
func prettyPrintJsonToCLI(outcome Outcome) {
	data, err := json.MarshalIndent(outcome, "", "\t")
	if err != nil {
		fmt.Println("Could not marshal the output: ", err.Error())
		os.Exit(1)
	}
	fmt.Println(string(data))
}

// prettyPrintOutcomeToCLI will print the probe outcome to the CLI, pretty-printed
func prettyPrintOutcomeToCLI(outcome Outcome) {
	fmt.Printf("%sRequest:\n", colorWhite)
	fmt.Printf("%sMethod:\t%s%s\n", colorCyan, colorReset, outcome.Requester.Method)
	fmt.Printf("%sURL:\t%s%s\n", colorCyan, colorReset, outcome.Requester.Url)
	fmt.Printf("%sT/Out:\t%s%s\n", colorCyan, colorReset, outcome.Requester.Timeout)
	fmt.Printf("%sResponse:\n", colorWhite)
	fmt.Printf("%sStatus:\t%s\n", colorCyan, statusInColor(outcome))
	fmt.Printf("%sSize:\t%s%s\n", colorCyan, colorReset, byteCountDecimal(outcome.Size))
	if outcome.Err != nil {
		fmt.Printf("%sError:\t%s\n", colorRed, outcome.Err)
	}
	fmt.Printf("%sMetrics:\n", colorWhite)
	fmt.Printf("%sDNS:\t%s%s\n", colorCyan, colorReset, outcome.Metrics.DNS)
	fmt.Printf("%sConn:\t%s%s\n", colorCyan, colorReset, outcome.Metrics.Conn)
	fmt.Printf("%sTLS:\t%s%s\n", colorCyan, colorReset, outcome.Metrics.TLS)
	fmt.Printf("%sTTFB:\t%s%s\n", colorCyan, colorReset, outcome.Metrics.TTFB)
	fmt.Printf("%sData:\t%s%s\n", colorCyan, colorReset, outcome.Metrics.Transfer)
	fmt.Printf("%sRT:\t%s%s\n", colorCyan, colorReset, outcome.Metrics.RT)
	if len(outcome.Checks) > 0 {
		fmt.Printf("%sAssertions:\n", colorWhite)
		for _, check := range outcome.Checks {
			if check.Success {
				fmt.Print(colorGreen)
			} else {
				fmt.Printf(colorRed)
			}
			fmt.Println(check.Assertion, "->", check.Output)
		}
	}
	fmt.Println(colorReset)
}

// statusInColor will print the status code in the right color
func statusInColor(outcome Outcome) string {
	if outcome.StatusCode < 300 {
		return colorGreen + strconv.Itoa(outcome.StatusCode)
	}
	if outcome.StatusCode < 400 {
		return colorYellow + strconv.Itoa(outcome.StatusCode)

	}
	return colorRed + strconv.Itoa(outcome.StatusCode)
}

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
