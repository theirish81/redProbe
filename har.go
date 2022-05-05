package main

import (
	"net/http"
	"net/url"
	"time"
)

// Har is the container of the HAR format output
type Har struct {
	Log Log `json:"log"`
}

// Log is the root of he HAR format output
type Log struct {
	Version string  `json:"version"`
	Creator Creator `json:"creator"`
	Entries []Entry `json:"entries"`
}

// Creator is the HAR creator information
type Creator struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Entry is one entry in the HAR file
type Entry struct {
	StartedDateTime time.Time     `json:"startedDateTime"`
	Time            int           `json:"time"`
	Request         EntryRequest  `json:"request"`
	Response        EntryResponse `json:"response"`
	Cache           interface{}   `json:"cache"`
	Timings         Timings       `json:"timings"`
}

// Timings is the recorded timings for a HAR entry
type Timings struct {
	Blocked int `json:"blocked"`
	DNS     int `json:"dns"`
	Connect int `json:"connect"`
	Send    int `json:"send"`
	Wait    int `json:"wait"`
	Receive int `json:"receive"`
	SSL     int `json:"ssl"`
}

// EntryRequest is the request of a HAR entry
type EntryRequest struct {
	Method      string         `json:"method"`
	HttpVersion string         `json:"httpVersion"`
	URL         string         `json:"url"`
	Headers     []EntryPair    `json:"headers"`
	HeadersSize int            `json:"headersSize"`
	Cookies     []interface{}  `json:"cookies"`
	QueryString []EntryPair    `json:"queryString"`
	PostData    *EntryPostData `json:"postData"`
	BodySize    int            `json:"bodySize"`
}

// EntryPair is a name/value entry for the HAR file
type EntryPair struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// EntryCookie is a cookie definition for the HAR file
type EntryCookie struct {
	Name     string    `json:"name"`
	Value    string    `json:"value"`
	Path     string    `json:"path"`
	Domain   string    `json:"domain"`
	Expires  time.Time `json:"expires"`
	HttpOnly bool      `json:"httpOnly"`
	Secure   bool      `json:"secure"`
}

// NewEntryCookie is the constructor for EntryCookie
func NewEntryCookie(cookie *http.Cookie) EntryCookie {
	return EntryCookie{Name: cookie.Name, Value: cookie.Value, Path: cookie.Path, Domain: cookie.Domain,
		Expires: cookie.Expires, HttpOnly: cookie.HttpOnly, Secure: cookie.Secure}
}

// EntryPostData is the container of post data for the HAR file
type EntryPostData struct {
	MimeType string `json:"mimeType"`
	Text     string `json:"text"`
}

// EntryResponse is the response definition for an HAR file entry
type EntryResponse struct {
	Status      int           `json:"status"`
	StatusText  string        `json:"statusText"`
	Headers     []EntryPair   `json:"headers"`
	RedirectURL string        `json:"redirectURL"`
	HttpVersion string        `json:"httpVersion"`
	Cookies     []EntryCookie `json:"cookies"`
	HeadersSize int           `json:"headersSize"`
	BodySize    int           `json:"bodySize"`
	Content     *EntryContent `json:"content"`
}

// EntryContent is the content definition for a HAR file entry
type EntryContent struct {
	Size     int    `json:"size"`
	MimeType string `json:"mimeType"`
	Text     string `json:"text"`
}

// toHar converts an array of "outcomes" to a Har object
func toHar(outcomes []Outcome) Har {
	log := Log{Creator: Creator{Name: "RedProbe", Version: "1.0.0"}}
	log.Version = "1.2"
	log.Entries = []Entry{}
	for _, o := range outcomes {
		entry := Entry{StartedDateTime: o.StartTime}
		entry.Time = int(o.Metrics.RT.Seconds())
		request := EntryRequest{Method: o.Requester.Method, URL: o.Requester.Url}
		request.HttpVersion = "HTTP/2.0"
		request.HeadersSize = -1
		request.Cookies = make([]interface{}, 0)
		request.Headers = make([]EntryPair, 0)
		for k, v := range o.Requester.Headers {
			request.Headers = append(request.Headers, EntryPair{Name: k, Value: v})
		}
		parsedUrl, _ := url.Parse(request.URL)
		request.QueryString = make([]EntryPair, 0)
		for k, v := range parsedUrl.Query() {
			request.QueryString = append(request.QueryString, EntryPair{Name: k, Value: v[0]})
		}
		if len(o.Requester.Body) > 0 {
			request.PostData = &EntryPostData{}
			request.PostData.Text = o.Requester.Body
			request.PostData.MimeType = o.Requester.getContentType()
			entry.Request.BodySize = len(o.Requester.Body)
		}
		entry.Request = request

		response := EntryResponse{}
		response.HeadersSize = -1
		response.HttpVersion = o.httpVersion
		response.Status = o.StatusCode
		response.StatusText = o.statusText
		response.Headers = make([]EntryPair, 0)
		for k, v := range o.Header {
			response.Headers = append(response.Headers, EntryPair{Name: k, Value: v[0]})
		}
		response.RedirectURL = o.Header.Get("Location")
		if len(o.bodyBytes) > 0 {
			response.Content = &EntryContent{}
			response.Content.Text = string(o.bodyBytes)
			response.Content.Size = len(o.bodyBytes)
			response.BodySize = response.Content.Size
			response.Content.MimeType = o.Header.Get("Content-Type")
		}
		response.Cookies = make([]EntryCookie, 0)
		if len(o.cookies) > 0 {
			for _, c := range o.cookies {
				response.Cookies = append(response.Cookies, NewEntryCookie(c))
			}
		}

		entry.Response = response
		entry.Cache = map[string]interface{}{}
		entry.Timings = Timings{Send: 0, Connect: int(o.Metrics.Conn.Milliseconds()),
			Receive: int(o.Metrics.Transfer.Milliseconds()), Blocked: 0, SSL: int(o.Metrics.TLS.Milliseconds()),
			Wait: int(o.Metrics.TTFB.Milliseconds()), DNS: int(o.Metrics.DNS.Milliseconds())}
		log.Entries = append(log.Entries, entry)
	}
	return Har{Log: log}
}
