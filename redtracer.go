package main

import (
	"crypto/tls"
	"net/http"
	"net/http/httptrace"
	"time"
)

// RedTracer will collect times of the events during an HTTP call
type RedTracer struct {
	clientTrace httptrace.ClientTrace
	connStart   time.Time
	connDone    time.Time
	dnsStart    time.Time
	dnsEnd      time.Time
	tlsStart    time.Time
	tlsEnd      time.Time
	firstByte   time.Time
	complete    time.Time
}

// newRedTracer is the constructor for RedTracer
func newRedTracer() *RedTracer {
	rt := RedTracer{}
	rt.clientTrace = httptrace.ClientTrace{
		DNSStart: func(info httptrace.DNSStartInfo) {
			rt.dnsStart = time.Now()
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			rt.dnsEnd = time.Now()
		},
		ConnectStart: func(network, addr string) {
			rt.connStart = time.Now()
		},
		ConnectDone: func(network, addr string, err error) {
			rt.connDone = time.Now()
		},
		TLSHandshakeStart: func() {
			rt.tlsStart = time.Now()
		},
		TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			rt.tlsEnd = time.Now()
		},
		GotFirstResponseByte: func() {
			rt.firstByte = time.Now()
		}}
	return &rt
}

// addContext adds the tracer context to the request
func (rt *RedTracer) addContext(req *http.Request) *http.Request {
	ctx := httptrace.WithClientTrace(req.Context(), &rt.clientTrace)
	return req.WithContext(ctx)
}

// stop needs to be invoked at the end of the request, once the body has been pulled. This method records the end of
// the conversation as there's no other way to know that
func (rt *RedTracer) stop() {
	rt.complete = time.Now()
}

// dns will return the DNS duration
func (rt *RedTracer) dns() time.Duration {
	return rt.dnsEnd.Sub(rt.dnsStart)
}

// conn will return the time to connect duration
func (rt *RedTracer) conn() time.Duration {
	return rt.connDone.Sub(rt.connStart)
}

// tls will return the TLS handshake duration
func (rt *RedTracer) tls() time.Duration {
	return positiveOrZero(rt.tlsEnd.Sub(rt.tlsStart))
}

// ttfb will return the Time-To-First-Byte duration, which is the time from the complete handshake and the first byte
func (rt *RedTracer) ttfb() time.Duration {
	if rt.tlsStart.IsZero() {
		return positiveOrZero(rt.firstByte.Sub(rt.connDone))
	}
	return positiveOrZero(rt.firstByte.Sub(rt.tlsEnd))
}

// transfer is the duration of the data transfer
func (rt *RedTracer) transfer() time.Duration {
	return positiveOrZero(rt.complete.Sub(rt.firstByte))
}

// positiveOrZero will return the provided duration if it's greater than zero, or zero otherwise
func positiveOrZero(duration time.Duration) time.Duration {
	if duration.Nanoseconds() > 0 {
		return duration
	}
	return 0 * time.Nanosecond
}
