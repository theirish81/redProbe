package main

import (
	"bytes"
	"encoding/json"
	"errors"
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

func (e *RedError) Error() string {
	return e.Err.Error()
}

// Requester is the agent performing the request
type Requester struct {
	Method      string            `json:"method" yaml:"method"`
	Url         string            `json:"url" yaml:"url"`
	Headers     map[string]string `json:"headers" yaml:"headers"`
	Body        string            `json:"body" yaml:"body"`
	Timeout     Duration          `json:"timeout" yaml:"timeout"`
	Assertions  []string          `json:"assertions" yaml:"assertions"`
	Annotations []string          `json:"annotations" yaml:"annotations"`
}

type Response struct {
	*http.Response
	*Outcome
	bodyBytes []byte
}

func (r *Response) JsonMap() map[string]interface{} {
	intFace := make(map[string]interface{})
	_ = json.Unmarshal(r.bodyBytes, &intFace)
	return intFace
}

func (r *Response) JsonArray() []interface{} {
	intFace := make([]interface{}, 0)
	_ = json.Unmarshal(r.bodyBytes, &intFace)
	return intFace
}

// Outcome is the result of the conversation
type Outcome struct {
	Requester   Requester    `json:"request"`
	Status      int          `json:"statusCode"`
	Size        int          `json:"size"`
	Metrics     Metrics      `json:"metrics"`
	Err         *RedError    `json:"error"`
	Annotations []Annotation `json:"annotations"`
	Checks      []Check      `json:"checks"`
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

type Annotation struct {
	Annotation string      `json:"annotation"`
	Text       interface{} `json:"text"`
}

// newRequester is the constructor for requester
func newRequester(method string, url string, headers map[string]string, body []byte, timeout Duration, assertions []string, annotations []string) Requester {
	return Requester{Method: method, Url: url, Headers: headers, Body: string(body), Timeout: timeout, Assertions: assertions, Annotations: annotations}
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
	outcome.Status = res.StatusCode
	res2 := Response{res, &outcome, bodyBytes}
	applyMetricsToOutcome(rt, &outcome)
	executeAnnotations(r.Annotations, &res2)
	executeAssertions(r.Assertions, &res2)
	return outcome
}

func executeAnnotations(annotations []string, response *Response) {
	for _, annotation := range annotations {
		env := map[string]interface{}{"Response": response, "Outcome": response}
		program, err := expr.Compile(annotation, expr.Env(env))
		if err != nil {
			response.Annotations = append(response.Annotations, Annotation{annotation, err.Error()})
			continue
		}
		result, err := expr.Run(program, env)
		response.Annotations = append(response.Annotations, Annotation{annotation, result})
	}
}

// executeAssertions will execute all assertions and store the results in outcome
func executeAssertions(assertions []string, response *Response) {
	for _, assertion := range assertions {
		env := map[string]interface{}{"Response": response, "Outcome": response}
		program, err := expr.Compile(assertion, expr.Env(env))
		if err != nil {
			response.Checks = append(response.Checks, Check{false, err.Error(), assertion})
			continue
		}
		result, err := expr.Run(program, env)
		if err != nil {
			response.Checks = append(response.Checks, Check{false, err.Error(), assertion})
			continue
		}
		switch v := result.(type) {
		case int:
			response.Checks = append(response.Checks, Check{v == 1, v, assertion})
		case bool:
			result = strconv.FormatBool(v)
			response.Checks = append(response.Checks, Check{v, v, assertion})
		case string:
			result = v
			response.Checks = append(response.Checks, Check{strings.ToLower(strings.TrimSpace(v)) == "ok", v, assertion})
		}
	}
}

// applyMetricsToOutcome takes the data from the tracer and applies them to the outcome
func applyMetricsToOutcome(rt *RedTracer, outcome *Outcome) {
	outcome.Metrics = Metrics{DNS: rt.dns(), TLS: rt.tls(), Conn: rt.conn(), TTFB: rt.ttfb(), Transfer: rt.transfer(), RT: rt.rt()}
}
