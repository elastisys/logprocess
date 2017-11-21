# logprocess
The logprocess repository is intended to contain programs for
processing log files.

For starters, it contains `apache2metric`, a simple program for web server logs
that follow the
[common log format](https://en.wikipedia.org/wiki/Common_Log_Format). The script
takes a sequence of apache request log file and converts them to a request count 
metric file that adheres to the `LINE FORMAT` below. The resulting metric file 
content is written to `stdout`. Requests in the apache log are counted over the 
specified sampling-interval and reported as an accumulated sum of requests at every
sampling point. The metric file can be used to ingest the metrics into a time-series
database such as InfluxDB.

The `LINE FORMAT` will produce an output similar to the following

    # time (ISO8601)          metric    value
    2012-12-31T23:00:00.000Z  reqcount  100
    2012-12-31T23:00:10.000Z  reqcount  150
    2012-12-31T23:00:20.000Z  reqcount  210

The metric name to use in the output as well as the sampling interval can
be passed to `apache2metric` via the `--metric-name` and `--sampling-interval`
options, respectively.


## Build
Build the binary with:

    go build -o apache2metric cmd/apache2metric/main.go


## Run

    ./apache2metric --metric-name=reqcount --sampling-interval=5s /path/to/logfile1 /path/to/logfile2

The program assumes that request log files (and their content) are ordered in
increasing order of time. It produces output (following the `LINE FORMAT` above) on
`stdout`.
