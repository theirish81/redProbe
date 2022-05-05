package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func TestCli(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	req := newRequester("GET", "https://www.google.com", map[string]string{"Accept": "text/html"}, []byte{},
		Duration{10 * time.Second}, false, []string{}, []string{})
	outcome := req.run()
	printToCli([]Outcome{outcome}, "console")
	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		outC <- buf.String()
	}()
	_ = w.Close()
	os.Stdout = old
	d2 := <-outC
	if !strings.Contains(d2, "https://www.google.com") ||
		!strings.Contains(d2, "DNS") {
		t.Error("ResponseOutcomeWrapper not coherent")
	}
}
