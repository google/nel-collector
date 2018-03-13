// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package collector

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ReportBatch is a collection of reports that should all be processed together.
// We will create a new batch for each upload that the collector receives.
// Certain processors might join batches together or split them up.
type ReportBatch struct {
	Reports []NelReport

	// When this batch was received by the collector
	Time time.Time

	// The IP address of the client that uploaded the batch of reports.  You can
	// typically assume that's the same IP address that was used for the original
	// requests.
	ClientIP string

	// The user agent of the client that uploaded the batch of reports.
	ClientUserAgent string
}

// A ReportProcessor implements one discrete processing step for handling
// uploaded reports.  There are several predefined processors, which you can use
// to filter or publish reports.  You can also implement custom annotation steps
// if you want to add additional data to a report before publishing.
type ReportProcessor interface {
	// ProcessReports handles a single batch of reports.  You have full control
	// over the contents of the batch; for instance, you can remove elements or
	// update their contents, if appropriate.
	ProcessReports(batch *ReportBatch)
}

// A ReportDumper is a ReportProcessor that prints out a summary of each report.
type ReportDumper struct {
	Writer io.Writer
}

// ProcessReports prints out a summary of each report in the batch.
func (d ReportDumper) ProcessReports(batch *ReportBatch) {
	time := batch.Time.UTC().Format("02/Jan/2006:15:04:05.000 -0700")
	for _, report := range batch.Reports {
		if report.ReportType == "network-error" {
			var result string
			if report.Type == "ok" || report.Type == "http.error" {
				result = strconv.Itoa(report.StatusCode)
			} else {
				result = report.Type
			}
			fmt.Fprintf(d.Writer, "%s - - [%s] \"GET %s\" %s -\n", batch.ClientIP, time, report.URL, result)
		} else {
			fmt.Fprintf(d.Writer, "%s - - [%s] \"GET %s\" <%s> -\n", batch.ClientIP, time, report.URL, report.ReportType)
		}
	}
}

// Clock lets you override how a pipeline assigns timestamps to each report.
// The default is to use time.Now; you can provide a custom implementation to
// get reproducible timestamps in test cases.
type Clock interface {
	Now() time.Time
}

type nowClock struct{}

func (c nowClock) Now() time.Time {
	return time.Now()
}

var defaultClock nowClock

// Pipeline is a series of processors that should be applied to each report that
// the collector receives.
type Pipeline struct {
	processors []ReportProcessor
	clock      Clock
}

// NewTestPipeline creates a Pipeline that will use a particular Clock to assign
// times to each report batch, instead of using time.Now.
func NewTestPipeline(clock Clock) *Pipeline {
	return &Pipeline{clock: clock}
}

// AddProcessor adds a new processor to the pipeline.
func (p *Pipeline) AddProcessor(processor ReportProcessor) {
	p.processors = append(p.processors, processor)
}

// ServeHTTP listens for POSTed report uploads, as defined by the Reporting
// spec, and runs all of the processors in the pipeline against each report.
func (p *Pipeline) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Must use POST to upload reports", http.StatusMethodNotAllowed)
		return
	}

	contentType := r.Header.Get("Content-Type")
	if contentType != "application/report" {
		http.Error(w, "Must use application/report to upload reports", http.StatusBadRequest)
		return
	}

	clock := p.clock
	if clock == nil {
		clock = defaultClock
	}

	var reports ReportBatch
	reports.Time = clock.Now()
	reports.ClientIP = strings.Split(r.RemoteAddr, ":")[0]
	reports.ClientUserAgent = r.Header.Get("User-Agent")
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&reports.Reports)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	for _, publisher := range p.processors {
		publisher.ProcessReports(&reports)
	}
	// 204 isn't an error, per-se, but this does the right thing.
	http.Error(w, "", http.StatusNoContent)
}
