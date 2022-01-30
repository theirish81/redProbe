# RedProbe
RedProbe is a simple HTTP probe that will return status code and a detailed breakdown of the request/response
performance metrics. The program also includes a simple assertion mechanism to validate whether the outcome of the
operation matches certain criteria.

## Status
[![CircleCI](https://circleci.com/gh/theirish81/redProbe/tree/master.svg?style=svg)](https://circleci.com/gh/theirish81/redProbe/tree/master)

## Download
Go to the [GitHub Releases Page](https://github.com/theirish81/redProbe/releases) for RedProbe and download the latest
release for your architecture.

## Run
You can run RedProbe in to ways:

### By providing all the parameters in the CLI
Here are the parameters:
```shell
 -A, --assertion=value  Assertion
 -f, --format=value     The output format, either
                        'console' or 'JSON' [console]
 -H, --header=value     The headers
 -t, --timeout=value    The request timeout [5s]
 -u, --url=value        The URL
 -X, --method=value     The method [GET]
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
  - Outcome.StatusCode == 200
  - Outcome.Size > 0
  - Outcome.Metrics.DNS.Milliseconds() < 200
  - Outcome.Metrics.RT.Seconds() < 2
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
  - Outcome.StatusCode == 200
  - Outcome.Size > 0
  - Outcome.Metrics.DNS.Milliseconds() < 200
  - Outcome.Metrics.RT.Seconds() < 2
---

url: https://github.com/theirish81/redProbe
timeout: 5s
headers:
  user-agent: redProbe/1
assertions:
  - Outcome.StatusCode == 200
  - Outcome.Size > 0
  - Outcome.Metrics.DNS.Milliseconds() < 200
  - Outcome.Metrics.RT.Seconds() < 2
```
