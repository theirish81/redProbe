package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/antonmedv/expr"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Duration is a wrapper around time.Duration to allow proper unmarshalling
type Duration struct {
	time.Duration
}

// UnmarshalJSON hides the time.Duration implementation
func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		d.Duration = time.Duration(value)
		return nil
	case string:
		var err error
		d.Duration, err = time.ParseDuration(value)
		if err != nil {
			return err
		}
		return nil
	default:
		return errors.New("invalid duration")
	}
}

// Requester is the agent performing the request
type Requester struct {
	Method     string            `json:"method" yaml:"method"`
	Url        string            `json:"url" yaml:"url"`
	Headers    map[string]string `json:"headers" yaml:"headers"`
	Body       string            `json:"body" yaml:"body"`
	Timeout    Duration          `json:"timeout" yaml:"timeout"`
	Format     string            `json:"format" yaml:"format"`
	Assertions []string          `json:"assertions" yaml:"assertions"`
}

// Outcome is the result of the conversation
type Outcome struct {
	Requester  Requester `json:"request"`
	StatusCode int       `json:"statusCode"`
	Metrics    Metrics   `json:"metrics"`
	Err        error     `json:"error"`
	Checks     []Check   `json:"checks"`
}

// isSuccess will return true when no errors happened during the call, and all assertions passed
func (o *Outcome) isSuccess() bool {
	if o.Err != nil {
		return false
	}
	for _, check := range o.Checks {
		if !check.Success {
			return false
		}
	}
	return true
}

// Metrics are the collected metrics
type Metrics struct {
	DNS      time.Duration `json:"DNS"`
	Conn     time.Duration `json:"conn"`
	TLS      time.Duration `json:"TLS"`
	TTFB     time.Duration `json:"TTFB"`
	Transfer time.Duration `json:"transfer"`
	RT       time.Duration `json:"rt"`
}

// Check is the result of an assertion execution
type Check struct {
	Success   bool        `json:"success"`
	Output    interface{} `json:"output"`
	Assertion string      `json:"assertion"`
}

// newRequester is the constructor for requester
func newRequester(method string, url string, headers map[string]string, body []byte, timeout Duration, assertions []string, format string) Requester {
	return Requester{Method: method, Url: url, Headers: headers, Body: string(body), Timeout: timeout, Assertions: assertions, Format: format}
}

// run performs the call
func (r *Requester) run() Outcome {
	outcome := Outcome{Requester: *r}
	request, _ := http.NewRequest(r.Method, r.Url, bytes.NewReader([]byte(r.Body)))
	for k, v := range r.Headers {
		request.Header.Set(k, v)
	}
	rt := newRedTracer()
	request = rt.addContext(request)
	client := http.Client{Timeout: r.Timeout.Duration}
	res, err := client.Do(request)
	if err != nil {
		outcome.Err = err
		applyMetricsToOutcome(rt, &outcome)
		return outcome
	}
	_, outcome.Err = io.Copy(ioutil.Discard, res.Body)
	if res.Body != nil {
		_ = res.Body.Close()
	}
	rt.stop()
	outcome.StatusCode = res.StatusCode
	applyMetricsToOutcome(rt, &outcome)
	executeAssertions(r.Assertions, &outcome)
	return outcome
}

// executeAssertions will execute all assertions and store the results in outcome
func executeAssertions(assertions []string, outcome *Outcome) {
	for _, assertion := range assertions {
		env := map[string]interface{}{"Outcome": *outcome}
		program, err := expr.Compile(assertion, expr.Env(env))
		if err != nil {
			outcome.Checks = append(outcome.Checks, Check{false, err.Error(), assertion})
			break
		}
		result, err := expr.Run(program, env)
		if err != nil {
			outcome.Checks = append(outcome.Checks, Check{false, err.Error(), assertion})
			break
		}
		switch v := result.(type) {
		case int:
			outcome.Checks = append(outcome.Checks, Check{v == 1, v, assertion})
		case bool:
			result = strconv.FormatBool(v)
			outcome.Checks = append(outcome.Checks, Check{v, v, assertion})
		case string:
			result = v
			outcome.Checks = append(outcome.Checks, Check{strings.ToLower(strings.TrimSpace(v)) != "ok", v, assertion})
		}
	}
}

// applyMetricsToOutcome takes the data from the tracer and applies them to the outcome
func applyMetricsToOutcome(rt *RedTracer, outcome *Outcome) {
	outcome.Metrics = Metrics{DNS: rt.dns(), TLS: rt.tls(), Conn: rt.conn(), TTFB: rt.ttfb(), Transfer: rt.transfer(), RT: rt.rt()}
}
