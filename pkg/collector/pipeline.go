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
	"context"
	"encoding/json"
	"net"
	"net/http"
	"time"
)

// A ReportProcessor implements one discrete processing step for handling
// uploaded reports.  There are several predefined processors, which you can use
// to filter or publish reports.  You can also implement custom annotation steps
// if you want to add additional data to a report before publishing.
type ReportProcessor interface {
	// ProcessReports handles a single batch of reports.  You have full control
	// over the contents of the batch; for instance, you can remove elements or
	// update their contents, if appropriate.
	ProcessReports(ctx context.Context, batch *ReportBatch)
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

// ContextGetter extracts a context from a http.Request. This allows for more
// complex logic for getting a context beyond r.Context()
type ContextGetter interface {
	Context(r *http.Request) context.Context
}

// defaultContextGetter implements a ContextGetter that gets the context
// contained directly within the request
type defaultContextGetter struct{}

// Context returns the context contained directly within the request.
func (d defaultContextGetter) Context(r *http.Request) context.Context {
	return r.Context()
}

var defaultClock nowClock

// Pipeline is a series of processors that should be applied to each report that
// the collector receives.
type Pipeline struct {
	ctxGetter  ContextGetter
	processors []ReportProcessor
	clock      Clock
}

// NewPipeline creates a new Pipeline that uses a particular clock.  For
// production pipelines, just instantiate the Pipeline type yourself
// (&Pipeline{}).
func NewPipeline(clock Clock) *Pipeline {
	return &Pipeline{ctxGetter: DefaultContextGetter{}, clock: clock}
}

// AddProcessor adds a new processor to the pipeline.
func (p *Pipeline) AddProcessor(processor ReportProcessor) {
	p.processors = append(p.processors, processor)
}

// SetContextGetter overrides the default (or current) ContextGetter with cg.
func (p *Pipeline) SetContextGetter(cg ContextGetter) {
	p.ctxGetter = cg
}

// ProcessReports extracts reports from a POST upload payload, as defined by the
// Reporting spec, and runs all of the processors in the pipeline against each
// report.
func (p *Pipeline) ProcessReports(ctx context.Context, w http.ResponseWriter, r *http.Request) *ReportBatch {
	if r.Method != "POST" {
		http.Error(w, "Must use POST to upload reports", http.StatusMethodNotAllowed)
		return nil
	}

	contentType := r.Header.Get("Content-Type")
	if contentType != "application/report" {
		http.Error(w, "Must use application/report to upload reports", http.StatusBadRequest)
		return nil
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return nil
	}

	clock := p.clock
	if clock == nil {
		clock = defaultClock
	}

	var reports ReportBatch
	reports.Time = clock.Now()
	reports.CollectorURL = *r.URL
	reports.ClientIP = host
	reports.ClientUserAgent = r.Header.Get("User-Agent")
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&reports.Reports)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return nil
	}

	for _, publisher := range p.processors {
		publisher.ProcessReports(ctx, &reports)
	}
	// 204 isn't an error, per-se, but this does the right thing.
	http.Error(w, "", http.StatusNoContent)
	return &reports
}

// serveCORS handles OPTIONS requests by allowing POST requests with a
// Content-Type header from any origin.
func serveCORS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Origin", "*")
}

// ServeHTTP handles POST report uploads, extracting the payload and handing it
// off to ProcessReports for processing.
func (p *Pipeline) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		serveCORS(w, r)
		return
	}
	ctx := p.ctxGetter.Context(r)
	p.ProcessReports(ctx, w, r)
}
