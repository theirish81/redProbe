package main

import (
	"testing"
)

func TestBasicHar(t *testing.T) {
	outcomes := make([]Outcome, 0)
	outcome := Outcome{StatusCode: 200, Metrics: Metrics{Transfer: 10, DNS: 20, Conn: 30, TTFB: 40}}
	outcome.Header = map[string][]string{"foo": {"bar"}}
	outcome.Size = 200
	outcomes = append(outcomes, outcome)
	har := toHar(outcomes)
	if len(har.Log.Entries) != 1 {
		t.Errorf("not enough entries")
	}
	entry := har.Log.Entries[0]
	if entry.Response.Status != 200 {
		t.Errorf("wrong status code")
	}
	if entry.Response.Headers[0].Name != "foo" || entry.Response.Headers[0].Value != "bar" {
		t.Errorf("wrong response headers")
	}
}
