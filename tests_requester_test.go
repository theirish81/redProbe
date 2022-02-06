package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestExecuteAssertions(t *testing.T) {
	outcome := Outcome{Status: 200, Metrics: Metrics{DNS: 10 * time.Second}}
	res := Response{&http.Response{StatusCode: 200}, &outcome, nil}
	executeAssertions([]string{
		"Response.StatusCode==200",
		"Response.Metrics.DNS.Seconds() > 10",
		"Response.Foo",
		"Response.StatusCode==200 ? \"OK\" : \"Nope\""}, &res)
	if !outcome.Checks[0].Success {
		t.Error("Assertion did not pass")
	}
	if outcome.Checks[1].Success {
		t.Error("Assertion did not pass")
	}
	if outcome.Checks[2].Success {
		t.Error("Syntactically wrong assertion passed")
	}
	if !outcome.Checks[3].Success || outcome.Checks[3].Output != "OK" {
		t.Error("Wrong OK return for assertion")
	}
	if outcome.isSuccess() {
		t.Error("This outcome should not be a success")
	}
}

func TestRequester(t *testing.T) {
	r := newRequester("GET", "https://www.example.com", map[string]string{"Accept": "text/html"}, []byte{},
		Duration{10 * time.Second}, []string{}, []string{})
	outcome := r.run()
	fmt.Println(outcome)
	if outcome.Status != 200 {
		t.Error("Status code is not correct")
	}
	if outcome.Metrics.DNS.Nanoseconds() <= 0 {
		t.Error("DNS is not correct")
	}
	if outcome.Metrics.TLS.Nanoseconds() <= 0 {
		t.Error("TLS is not correct")
	}
	if outcome.Metrics.Conn.Nanoseconds() <= 0 {
		t.Error("Conn is not correct")
	}
	if outcome.Metrics.Transfer.Nanoseconds() <= 0 {
		t.Error("Transfer is not correct")
	}
}

func TestJsonConversion(t *testing.T) {
	j1 := []byte("{\"foo\":\"bar\"}")
	res := Response{&http.Response{StatusCode: 200}, &Outcome{}, j1}
	executeAssertions([]string{"Response.JsonMap().foo==\"bar\""}, &res)
	if !res.Outcome.Checks[0].Success {
		t.Error("Json conversion assertion did not work")
	}
	j2 := []byte("[{\"foo\":\"bar\"}]")
	res = Response{&http.Response{StatusCode: 200}, &Outcome{}, j2}
	executeAssertions([]string{"Response.JsonArray()[0].foo==\"bar\""}, &res)
	if !res.Outcome.Checks[0].Success {
		t.Error("Json conversion assertion did not work")
	}
}

func TestLoadConfig(t *testing.T) {
	req := requesterFromConfig("sample_calls/example.yaml")
	if req[0].Timeout.Duration.Seconds() != 5 {
		t.Error("Could not parse duration from config file")
	}
}
