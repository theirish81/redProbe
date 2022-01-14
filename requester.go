package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
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
	Method  string            `json:"method" yaml:"method"`
	Url     string            `json:"url" yaml:"url"`
	Headers map[string]string `json:"headers" yaml:"headers"`
	Body    string            `json:"body" yaml:"body"`
	Timeout Duration          `json:"timeout" yaml:"timeout"`
	Format  string            `json:"format" yaml:"format"`
}

// Outcome is the result of the conversation
type Outcome struct {
	Requester  Requester `json:"request"`
	StatusCode int       `json:"statusCode"`
	Metrics    Metrics   `json:"metrics"`
	Err        error     `json:"error"`
}

// Metrics are the collected metrics
type Metrics struct {
	DNS      time.Duration `json:"DNS"`
	Conn     time.Duration `json:"conn"`
	TLS      time.Duration `json:"TLS"`
	TTFB     time.Duration `json:"TTFB"`
	Transfer time.Duration `json:"transfer"`
}

// newRequester is the constructor for requester
func newRequester(method string, url string, headers map[string]string, body []byte, timeout Duration, format string) Requester {
	return Requester{Method: method, Url: url, Headers: headers, Body: string(body), Timeout: timeout, Format: format}
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
		r.applyMetricsToOutcome(rt, &outcome)
		return outcome
	}
	_, outcome.Err = io.Copy(ioutil.Discard, res.Body)
	if res.Body != nil {
		_ = res.Body.Close()
	}
	rt.stop()
	outcome.StatusCode = res.StatusCode
	r.applyMetricsToOutcome(rt, &outcome)
	return outcome
}

// applyMetricsToOutcome takes the data from the tracer and applies them to the outcome
func (r *Requester) applyMetricsToOutcome(rt *RedTracer, outcome *Outcome) {
	outcome.Metrics = Metrics{DNS: rt.dns(), TLS: rt.tls(), Conn: rt.conn(), TTFB: rt.ttfb(), Transfer: rt.transfer()}
}
