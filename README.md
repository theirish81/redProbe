# RedProbe
RedProbe is a simple HTTP probe that will return status code and a detailed breakdown of the request/response
performance metrics. The program also includes a simple assertion mechanism to validate whether the outcome of the
operation matches certain criteria.

## Status
[![CircleCI](https://circleci.com/gh/theirish81/redProbe/tree/master.svg?style=svg)](https://circleci.com/gh/theirish81/redProbe/tree/master)

The software is stable. Due to the more advanced nature of *NIX terminals, the output is remarkably better in Linux and
OSX, but it does work acceptably well on Windows as well.

## Download
Go to the [GitHub Releases Page](https://github.com/theirish81/redProbe/releases) for RedProbe and download the latest
release for your architecture.

## Run
You can run RedProbe in to ways:

### By providing all the parameters in the CLI
Here are the parameters:
```shell
 -a, --annotation=value  Annotation
 -A, --assertion=value   Assertion
 -c, --config=value      Path to a config file
 -f, --format=value      The output format, either
                         'console', 'JSON' or 'HAR' [console]
 -H, --header=value      The headers
 -s, --skip-ssl          Skips SSL validation
 -t, --timeout=value     The request timeout [5s]
 -u, --url=value         The URL
 -X, --method=value      The method [GET]
```
The bare minimum request is done by providing a URL with `-u` or `--url=` as in:
```shell
./redprobe -u https://www.example.com
```

Headers are provided as couples of key/value pairs, separated by a colon, as in:
```shell
./redprobe -u https://www.example.com -H 'Accept:text/html' -H 'Key:ABC123'
```

### By providing a YAML configuration file
You can provide a configuration file a substitute of all the command line parameters, as in:
```yaml
url: https://www.example.com
timeout: 5s
headers:
  user-agent: redChecker/1
assertions:
  - Response.StatusCode == 200
  - Response.Size > 0
  - Response.Metrics.DNS.Milliseconds() < 200
  - Response.Metrics.RT.Seconds() < 2
```

### By providing a multi-document YAML configuration file
You can structure your YAML to contain multiple configuration files. If you do, RedProbe will execute all calls
in a sequence. As in:
```yaml
url: https://www.example.com
timeout: 5s
headers:
  user-agent: redProbe/1
assertions:
  - Response.StatusCode == 200
  - Response.Size > 0
  - Response.Metrics.DNS.Milliseconds() < 200
  - Response.Metrics.RT.Seconds() < 2
---

url: https://github.com/theirish81/redProbe
timeout: 5s
headers:
  user-agent: redProbe/1
assertions:
  - Response.StatusCode == 200
  - Response.Size > 0
  - Response.Metrics.DNS.Milliseconds() < 200
  - Response.Metrics.RT.Seconds() < 2
```

## Assertions and annotations

### Assertions
Optionally, you can add assertions as shown in the examples. The purpose of assertions is to set expectations about the
response and signal when those expectations are not met. If assertions fail, the program will return with a non-zero
status code. You can add an `assertions` block in the configuration file or add multiple `-A` arguments in the CLI.
Assertions will be considered a pass if they return either `true`, `ok`, or `1`.


### Annotations
Optionally, you can add annotations as shown in the examples. The purpose of annotations is to annotate the outcome with
values extracted from the response, for debugging purposes.  You can add an `annotations` block in the configuration file
or add multiple `-a` arguments in the CLI.

### Syntax
The root object of all annotations and assertions is `Response` (mind the capital R).

The base sub-items are:
* `StatusCode`: an integer representing the response status code
* `Size`: an integer representing the response size
* `IpAddress`: a string representing the target IP address
The structured sub-items are:
* `Metrics`: an object containing the metrics
    * `Conn`: the duration of the connection phase
    * `DNS`: the duration of the DNS resolution
    * `TLS`: the duration of the TLS handshake
    * `TTFB`: time to first byte
    * `Transfer`: the data transfer time
    * `RT`: round-trip time
  
  Each metric can be converted into a numerical representation by appending `.Seconds()`, `.Milliseconds()`, `.Nanoseconds()`
  as in: `Response.Metrics.DNS.Milliseconds()
* `Header`: a collection of response headers. Each header can be accessed by invoking:
  * `Get(headerName)`: will return the value of the header with the given name
* `JsonMap()`: trusting that the response body is a JSON object, the method will parse it and return a map
* `JsonArray()`: trusting that the response body is a JSON array, will method will parse it and return an array

#### Assertions examples
`Response.Metrics.DNS.Milliseconds() < 200`: pass if the DNS resolution time is less than 200 milliseconds
`Response.JsonMap().id == 1`: pass it the JSON object in the response has an ID field that is equal to 1

#### Annotations examples
`response.Header.Get("content-type")`: print the response content type header