package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/pborman/getopt/v2"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

//main is the main function. It does all the parameter parsing and triggers the execution of the probe
func main() {
	method := getopt.StringLong("method", 'X', "GET", "The method")
	url := getopt.StringLong("url", 'u', "", "The URL")
	headers := getopt.ListLong("header", 'H', "The headers")
	timeout := getopt.StringLong("timeout", 't', "5s", "The request timeout")
	format := getopt.StringLong("format", 'f', "console", "The output format, either 'console', 'JSON' or 'HAR'")
	assertions := getopt.ListLong("assertion", 'A', "Assertion")
	annotations := getopt.ListLong("annotation", 'a', "Annotation")
	config := getopt.StringLong("config", 'c', "", "Path to a config file")
	skipSSL := getopt.BoolLong("skip-ssl", 's', "Skips SSL validation")
	getopt.HelpColumn = 50
	getopt.Parse()
	requesters := make([]Requester, 0)
	if *config != "" {
		requesters = requesterFromConfig(*config)
	} else {
		requesters = append(requesters, requesterFromCli(*method, *url, *headers, readBody(), *timeout, *skipSSL, *assertions, *annotations))
	}
	outcomes := make([]Outcome, 0)
	if format != nil {
		*format = strings.ToLower(*format)
	}
	for _, requester := range requesters {
		requester.keepResponse = *format == "har"
		outcome := requester.run()
		outcomes = append(outcomes, outcome)
	}
	printToCli(outcomes, *format)
	for _, outcome := range outcomes {
		if !outcome.isSuccess() {
			os.Exit(1)
		}
	}
}

// readBody reads the request body from the standard input, if there's any
func readBody() []byte {
	var body []byte
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		reader := bufio.NewReader(os.Stdin)
		body, _ = ioutil.ReadAll(reader)
	}
	return body
}

// requesterFromConfig runs the CLI probe pulling the settings from a configuration file
func requesterFromConfig(path string) []Requester {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Println("Error reading the configuration file: ", err.Error())
		os.Exit(1)
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	requesters := make([]Requester, 0)
	for err == nil {
		req := newRequester("GET", "", make(map[string]string), make([]byte, 0), Duration{5 * time.Second}, false, []string{}, []string{})
		err = decoder.Decode(&req)
		if err != nil {
			break
		}
		requesters = append(requesters, req)
	}
	if err.Error() != "EOF" {
		fmt.Println("Error reading configuration file: ", err.Error())
		os.Exit(1)
	}
	return requesters
}

// requesterFromCli runs the command line probe using the parameters passed in the command line
func requesterFromCli(method string, urlString string, headers []string, body []byte, timeout string, skipSSL bool,
	assertions []string, annotations []string) Requester {
	if urlString == "" {
		getopt.PrintUsage(os.Stdout)
		os.Exit(1)
	}
	d, err := time.ParseDuration(timeout)
	if err != nil {
		fmt.Println("Could not parse timeout")
		os.Exit(1)
	}
	return newRequester(strings.ToUpper(method), urlString, arrayToMap(headers), body, Duration{d}, skipSSL,
		assertions, annotations)
}

// arrayToMap turns an array of colon-separated strings into a map
func arrayToMap(headers []string) map[string]string {
	data := map[string]string{}
	for _, h := range headers {
		subs := strings.SplitN(h, ":", 2)
		if len(subs) == 2 {
			data[strings.TrimSpace(subs[0])] = strings.TrimSpace(subs[1])
		}
	}
	return data
}
