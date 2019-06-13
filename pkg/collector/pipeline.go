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
	"errors"
	"fmt"
	"net/http"
	"sync"
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

var defaultClock nowClock

// Pipeline is a series of processors that should be applied to each report that
// the collector receives. It uses a fixed number of workers to process the reports
// and a fixed sized queue that the workers read from. If the queue fills, reports
// are dropped. Pipeline{} is not a usable instance, use NewPipeline for production
// and NewTestPipeline* in tests.
type Pipeline struct {
	processors []ReportProcessor
	clock      Clock
	c          chan *ReportBatch
	wg         *sync.WaitGroup
}

// NewPipeline creates a new Pipeline with a specified buffer size
// and number of workers.
func NewPipeline(bufferSize int64, numWorkers int) *Pipeline {
	return setupPipeline(context.Background(), nil, bufferSize, numWorkers)
}

const defaultBufferSize = 1000
const defaultNumWorkers = 10

// NewTestPipeline creates a new Pipeline with a specified clock.
// This should only be used for testing.
func NewTestPipeline(clock Clock) *Pipeline {
	return NewTestPipelineWithBuffer(clock, defaultBufferSize)
}

// NewTestPipelineWithBuffer creates a new Pipeline with a specified buffer size and clock.
// This should only be used for testing.
func NewTestPipelineWithBuffer(clock Clock, bufferSize int64) *Pipeline {
	return setupPipeline(context.Background(), clock, bufferSize, defaultNumWorkers)
}

func setupPipeline(ctx context.Context, clock Clock, bufferSize int64, numWorkers int) *Pipeline {
	p := &Pipeline{
		clock: clock,
		c:     make(chan *ReportBatch, bufferSize),
		wg:    &sync.WaitGroup{},
	}
	for i := 0; i < numWorkers; i++ {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for reports := range p.c {
				for _, publisher := range p.processors {
					publisher.ProcessReports(ctx, reports)
				}
			}
		}()
	}
	return p
}

// AddProcessor adds a new processor to the pipeline.
func (p *Pipeline) AddProcessor(processor ReportProcessor) {
	p.processors = append(p.processors, processor)
}

// ErrDropped is returned from ProcessReports when the queue is full and the report is dropped.
var ErrDropped = errors.New("queue full, report dropped")

// ProcessReports extracts reports from a POST upload payload, as defined by the
// Reporting spec, and runs all of the processors in the pipeline against each
// report. Returns ErrDropped if the request was dropped due to a full queue and nil
// on success. All other errors indicate something wrong with the request.
func (p *Pipeline) ProcessReports(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	if r.Method != "POST" {
		http.Error(w, "Must use POST to upload reports", http.StatusMethodNotAllowed)
		return fmt.Errorf("Must use POST to upload reports")
	}

	contentType := r.Header.Get("Content-Type")
	if contentType != "application/reports+json" {
		http.Error(w, "Must use application/reports+json to upload reports", http.StatusBadRequest)
		return fmt.Errorf("Must use application/reports+json to upload reports")
	}

	clock := p.clock
	if clock == nil {
		clock = defaultClock
	}

	reports, err := NewReportBatch(r, clock)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	// 204 isn't an error, per-se, but this does the right thing.
	http.Error(w, "", http.StatusNoContent)

	select {
	case p.c <- reports:
		return nil
	default:
		return ErrDropped
	}
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
	ctx := r.Context()
	p.ProcessReports(ctx, w, r)
}

// Close stops the processing, such that anything in the queue
// gets processed, but nothing is added. It then waits until all
// processing workers have completed. All calls to ProcessReports
// must complete before Close is called, otherwise it will cause
// a panic.
func (p *Pipeline) Close() {
	close(p.c)
	p.wg.Wait()
}
