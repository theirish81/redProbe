package main

import (
	"fmt"
	"testing"
	"time"
)

func TestExecuteAssertions(t *testing.T) {
	outcome := Outcome{StatusCode: 200, Metrics: Metrics{DNS: 10 * time.Second}}
	executeAssertions([]string{
		"Outcome.StatusCode==200",
		"Outcome.Metrics.DNS.Seconds() > 10",
		"Outcome.Foo",
		"Outcome.StatusCode==200 ? \"OK\" : \"Nope\""}, &outcome)
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
		Duration{10 * time.Second}, false, []string{}, []string{})
	outcome := r.run()
	fmt.Println(outcome)
	if outcome.StatusCode != 200 {
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
	res := Outcome{StatusCode: 200, bodyBytes: j1}
	executeAssertions([]string{"Outcome.JsonMap().foo==\"bar\""}, &res)
	if !res.Checks[0].Success {
		t.Error("Json conversion assertion did not work")
	}
	j2 := []byte("[{\"foo\":\"bar\"}]")
	res = Outcome{StatusCode: 200, bodyBytes: j2}
	executeAssertions([]string{"Response.JsonArray()[0].foo==\"bar\""}, &res)
	if !res.Checks[0].Success {
		t.Error("Json conversion assertion did not work")
	}
}

func TestLoadConfig(t *testing.T) {
	req := requesterFromConfig("sample_calls/example.yaml")
	if req[0].Timeout.Duration.Seconds() != 5 {
		t.Error("Could not parse duration from config file")
	}
}
