package main

import (
	"bufio"
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
	format := getopt.StringLong("format", 'f', "console", "The output format, either 'console' or 'JSON'")
	assertions := getopt.ListLong("assertion", 'A', "Assertion")
	config := getopt.StringLong("config", 'c', "", "Path to a config file")
	getopt.HelpColumn = 50
	getopt.Parse()
	var requester Requester
	if *config != "" {
		requester = requesterFromConfig(*config)
	} else {
		requester = requesterFromCli(*method, *url, *headers, readBody(), *timeout, *assertions, *format)
	}
	outcome := requester.run()
	printToCli(outcome)
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
func requesterFromConfig(path string) Requester {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Println("Error reading the configuration file: ", err.Error())
		os.Exit(1)
	}
	req := newRequester("GET", "", make(map[string]string), make([]byte, 0), Duration{5 * time.Second}, []string{}, "console")
	err = yaml.Unmarshal(data, &req)
	if err != nil {
		fmt.Println("Error reading the configuration file: ", err.Error())
		os.Exit(1)
	}
	return req
}

// requesterFromCli runs the command line probe using the parameters passed in the command line
func requesterFromCli(method string, urlString string, headers []string, body []byte, timeout string, assertions []string, format string) Requester {
	if urlString == "" {
		getopt.PrintUsage(os.Stdout)
		os.Exit(1)
	}
	d, err := time.ParseDuration(timeout)
	if err != nil {
		fmt.Println("Could not parse timeout")
		os.Exit(1)
	}
	return newRequester(strings.ToUpper(method), urlString, arrayToMap(headers), body, Duration{d}, assertions, strings.ToLower(format))
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
