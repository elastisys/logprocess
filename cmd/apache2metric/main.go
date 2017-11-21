package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

var (
	metricName       string
	samplingInterval time.Duration
)

const description string = `The script takes a sequence of apache request log
files (in increasing order of time) and downsamples them. The output, which is
written to stdout, represents an increasing request count that follows the 
LINE FORMAT described below. Requests in the apache log are counted over the 
specified sampling-interval and reported as an accumulated sum of requests at
every sampling point. The metric file can be used to ingest the metrics into
a time-series database such as InfluxDB.
	
The LINE FORMAT looks as follows:

  # time (ISO8601)          metric    value
  2012-12-31T23:00:00.000Z  reqcount  100
  2012-12-31T23:00:10.000Z  reqcount  150
  2012-12-31T23:00:20.000Z  reqcount  210		  
`

func dieWithError(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
	fmt.Fprintln(os.Stderr, "")
	flag.Usage()
	os.Exit(1)

}

func init() {
	flag.StringVar(&metricName, "metric-name", "reqcount",
		"The metric name to use in the output.")
	flag.DurationVar(&samplingInterval, "sampling-interval", 10*time.Second,
		"The sampling interval to use between reported request count metric values in the output file.")
}

// Parses out hte request time from an apache log entry, typically of form:
//
//     127.0.0.1 - - [10/Oct/2000:13:55:36 -0700] "GET / HTTP/1.0" 200 232
//
func extractRequestTime(line string) (*time.Time, error) {
	timeStart := strings.Index(line, "[")
	timeEnd := strings.Index(line, "]")
	if timeStart < 0 || timeEnd < 0 {
		return nil, fmt.Errorf("line does not contain a brace-enclosed timestamp: %s", line)
	}
	if timeStart > timeEnd {
		return nil, fmt.Errorf("line does not contain a brace-enclosed timestamp: %s", line)
	}
	time, err := time.Parse("02/Jan/2006:15:04:05 -0700", line[timeStart+1:timeEnd])
	return &time, err

}

func main() {
	flag.Parse()

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s:\n", description)
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS] FILE ...\n", os.Args[0])
		flag.PrintDefaults()
	}

	if len(flag.Args()) < 1 {
		dieWithError("error: no apache log files given")
	}
	logFilePaths := flag.Args()[0:]
	logFileReaders := []io.Reader{}
	for _, logFilePath := range logFilePaths {
		logFile, err := os.Open(logFilePath)
		if err != nil {
			dieWithError("error: %s", err)
		}
		defer logFile.Close()
		logFileReaders = append(logFileReaders, logFile)
	}
	multiReader := io.MultiReader(logFileReaders...)

	writer := bufio.NewWriter(os.Stdout)
	defer writer.Flush()
	scanner := bufio.NewScanner(multiReader)
	// request count so far
	reqCount := 0
	lineno := 1
	// tracks when the last request count observation was made
	var lastSampleTime *time.Time

	for scanner.Scan() {
		line := scanner.Text()
		t, err := extractRequestTime(strings.TrimSpace(line))
		if err != nil {
			fmt.Printf("line %d: failed to extract timestamp: %s", lineno, err)
			os.Exit(1)
		}

		if lastSampleTime == nil {
			// set last_sample_time on time of first request. from now on,
			// a metric will be reported every sampling_interval seconds.
			lastSampleTime = t
		}

		timeSinceSample := t.Unix() - lastSampleTime.Unix()
		if timeSinceSample > int64(samplingInterval.Seconds()) {
			sampleTime := time.Unix(lastSampleTime.Unix()+int64(samplingInterval.Seconds()), 0)
			writer.WriteString(fmt.Sprintf("%s  %s  %d\n", sampleTime.Format(time.RFC3339), metricName, reqCount))
			lastSampleTime = &sampleTime
		}

		lineno++
		reqCount++
	}
}
