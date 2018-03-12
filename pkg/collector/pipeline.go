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
)

// ReportBatch is a collection of reports that should all be processed together.
// We will create a new batch for each upload that the collector receives.
// Certain processors might join batches together or split them up.
type ReportBatch struct {
	Reports []NelReport
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
	for _, report := range batch.Reports {
		fmt.Fprintf(d.Writer, "[%s] %s\n", report.Type, report.URL)
	}
}

// Pipeline is a series of processors that should be applied to each report that
// the collector receives.
type Pipeline struct {
	processors []ReportProcessor
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

	var reports ReportBatch
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
