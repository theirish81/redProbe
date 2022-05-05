package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/antonmedv/expr"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Duration is a wrapper around time.Duration to allow proper unmarshalling
type Duration struct {
	time.Duration
}

// UnmarshalYAML hides the time.Duration implementation
func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var v interface{}
	if err := unmarshal(&v); err != nil {
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

// RedError is a wrapper for Error so that marshalling is easier and automatic
type RedError struct {
	Err error
}

// MarshalJSON will marshall the error message into JSON
func (e *RedError) MarshalJSON() (b []byte, err error) {
	if e.Err == nil {
		return nil, nil
	}
	return json.Marshal(e.Err.Error())
}

// Error returns the rror in string form
func (e *RedError) Error() string {
	return e.Err.Error()
}

// Requester is the agent performing the request
type Requester struct {
	Method       string            `json:"method" yaml:"method"`
	Url          string            `json:"url" yaml:"url"`
	Headers      map[string]string `json:"headers" yaml:"headers"`
	Body         string            `json:"body" yaml:"body"`
	Timeout      Duration          `json:"timeout" yaml:"timeout"`
	Assertions   []string          `json:"assertions" yaml:"assertions"`
	Annotations  []string          `json:"annotations" yaml:"annotations"`
	SkipSSL      bool              `json:"skipSSL" yaml:"skipSSL"`
	keepResponse bool
}

// Outcome is the result of the conversation
type Outcome struct {
	Requester   Requester    `json:"request"`
	StartTime   time.Time    `json:"startTime"`
	IpAddress   string       `json:"ip_address"`
	StatusCode  int          `json:"statusCode"`
	Size        int          `json:"size"`
	Metrics     Metrics      `json:"metrics"`
	Err         *RedError    `json:"error"`
	Annotations []Annotation `json:"annotations"`
	Checks      []Check      `json:"checks"`

	bodyBytes   []byte
	Header      http.Header `json:"-"`
	httpVersion string
	statusText  string
	cookies     []*http.Cookie
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

// JsonMap assumes the response body is a JSON and converts it into a Map
func (o *Outcome) JsonMap() map[string]interface{} {
	intFace := make(map[string]interface{})
	_ = json.Unmarshal(o.bodyBytes, &intFace)
	return intFace
}

// JsonArray assumes the response body is a JSON and converts it into an Array
func (o *Outcome) JsonArray() []interface{} {
	intFace := make([]interface{}, 0)
	_ = json.Unmarshal(o.bodyBytes, &intFace)
	return intFace
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

// Annotation is the result of an annotation execution
type Annotation struct {
	Annotation string      `json:"annotation"`
	Text       interface{} `json:"text"`
}

// newRequester is the constructor for requester
func newRequester(method string, url string, headers map[string]string, body []byte, timeout Duration, skipSSL bool,
	assertions []string, annotations []string) Requester {
	return Requester{Method: method, Url: url, Headers: headers, Body: string(body), Timeout: timeout, SkipSSL: skipSSL,
		Assertions: assertions, Annotations: annotations}
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
	transport := &http.Transport{
		MaxIdleConnsPerHost: 0,
	}
	if r.SkipSSL {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	client := http.Client{Timeout: r.Timeout.Duration, Transport: transport}
	outcome.StartTime = time.Now()
	res, err := client.Do(request)
	if err != nil {
		outcome.Err = &RedError{err}
		applyMetricsToOutcome(rt, &outcome)
		return outcome
	}
	bodyBytes, err := io.ReadAll(res.Body)
	outcome.Size = len(bodyBytes)
	if err != nil {
		outcome.Err = &RedError{err}
	}
	if res.Body != nil {
		_ = res.Body.Close()
	}
	rt.stop()
	outcome.StatusCode = res.StatusCode
	outcome.IpAddress = rt.ipAddress
	outcome.bodyBytes = bodyBytes
	outcome.Header = res.Header
	outcome.httpVersion = fmt.Sprintf("HTTP/%d.%d", res.ProtoMajor, res.ProtoMinor)
	outcome.statusText = res.Status
	outcome.cookies = res.Cookies()
	applyMetricsToOutcome(rt, &outcome)
	executeAnnotations(r.Annotations, &outcome)
	executeAssertions(r.Assertions, &outcome)
	if !r.keepResponse {
		outcome.Header = nil
		outcome.bodyBytes = nil
		outcome.cookies = nil
	}
	return outcome
}

// getContentType retrieves the content type from the request headers
func (r *Requester) getContentType() string {
	if res, ok := r.Headers["Content-Type"]; ok {
		return res
	}
	return r.Headers["content-type"]
}

// executeAnnotations will execute the annotations and store the results in outcome
func executeAnnotations(annotations []string, outcome *Outcome) {
	for _, annotation := range annotations {
		env := map[string]interface{}{"Response": outcome, "Outcome": outcome}
		program, err := expr.Compile(annotation, expr.Env(env))
		if err != nil {
			outcome.Annotations = append(outcome.Annotations, Annotation{annotation, err.Error()})
			continue
		}
		result, err := expr.Run(program, env)
		outcome.Annotations = append(outcome.Annotations, Annotation{annotation, result})
	}
}

// executeAssertions will execute all assertions and store the results in outcome
func executeAssertions(assertions []string, outcome *Outcome) {
	for _, assertion := range assertions {
		env := map[string]interface{}{"Response": outcome, "Outcome": outcome}
		program, err := expr.Compile(assertion, expr.Env(env))
		if err != nil {
			outcome.Checks = append(outcome.Checks, Check{false, err.Error(), assertion})
			continue
		}
		result, err := expr.Run(program, env)
		if err != nil {
			outcome.Checks = append(outcome.Checks, Check{false, err.Error(), assertion})
			continue
		}
		switch v := result.(type) {
		case int:
			outcome.Checks = append(outcome.Checks, Check{v == 1, v, assertion})
		case bool:
			result = strconv.FormatBool(v)
			outcome.Checks = append(outcome.Checks, Check{v, v, assertion})
		case string:
			result = v
			outcome.Checks = append(outcome.Checks, Check{strings.ToLower(strings.TrimSpace(v)) == "ok", v, assertion})
		}
	}
}

// applyMetricsToOutcome takes the data from the tracer and applies them to the outcome
func applyMetricsToOutcome(rt *RedTracer, outcome *Outcome) {
	outcome.Metrics = Metrics{DNS: rt.dns(), TLS: rt.tls(), Conn: rt.conn(), TTFB: rt.ttfb(), Transfer: rt.transfer(), RT: rt.rt()}
}
