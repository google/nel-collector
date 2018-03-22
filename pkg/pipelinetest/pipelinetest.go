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

// Package pipelinetest contains helper utilities for constructing test cases
// that exercise the components of a NEL collector pipeline.
package pipelinetest

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/nel-collector/pkg/collector"
)

type simulatedClock struct {
	currentTime time.Time
}

func newSimulatedClock() *simulatedClock {
	return &simulatedClock{currentTime: time.Unix(0, 0).UTC()}
}

func (c simulatedClock) Now() time.Time {
	return c.currentTime
}

// TestPipeline is a wrapper around Pipeline that is easier to use in test
// cases.  It uses a simulated clock, giving you reproducible timestamps in test
// output, and has helper methods for "uploading" a report payload.
type TestPipeline struct {
	*collector.Pipeline
	remoteAddr string
}

// NewTestPipeline creates a Pipeline that will use a particular Clock to assign
// times to each report batch, instead of using time.Now.
func NewTestPipeline(remoteAddr string) *TestPipeline {
	return &TestPipeline{collector.NewPipeline(newSimulatedClock()), remoteAddr}
}

// HandleCustomRequest processes a report payload as if it were uploaded using
// the given HTTP method and MIME type.  This is useful for test cases where you
// want to verify that uploads that don't conform to the Reporting spec are
// handled properly.
func (p *TestPipeline) HandleCustomRequest(t *testing.T, method, mimeType string, payload []byte) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, "https://example.com/upload/", bytes.NewReader(payload))
	request.Header.Add("Content-Type", mimeType)
	if p.remoteAddr != "" {
		request.RemoteAddr = p.remoteAddr
	}
	var response httptest.ResponseRecorder
	p.ServeHTTP(&response, request)
	return &response
}

// HandleRequest processes a report payload as if it were uploaded as required
// by the Reporting spec.  We assume that the payload is valid; if the pipeline
// doesn't return a 204 ("success with no response content"), we return an
// error.
func (p *TestPipeline) HandleRequest(t *testing.T, payload []byte) error {
	response := p.HandleCustomRequest(t, "POST", "application/report", payload)
	if response.Code != http.StatusNoContent {
		return fmt.Errorf("Incorrect status code: got %d, wanted %d", response.Code, http.StatusNoContent)
	}
	return nil
}

// StashReports is a pipeline processor that saves a copy of the report batch
// into some other variable under your control.  You can use this to verify the
// contents of the batch after the pipeline is done.
type StashReports struct {
	Dest *collector.ReportBatch
}

// ProcessReports saves a copy of the report batch into some other variable
// under your control.
func (s StashReports) ProcessReports(batch *collector.ReportBatch) {
	*s.Dest = *batch
}
