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
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

// Helpers

type simulatedClock struct {
	currentTime time.Time
}

func newSimulatedClock() *simulatedClock {
	return &simulatedClock{currentTime: time.Unix(0, 0).UTC()}
}

func (c simulatedClock) Now() time.Time {
	return c.currentTime
}

type testPipeline struct {
	name      string
	ipVersion string
	pipeline  *Pipeline
}

func newTestPipeline(name, ipVersion string) *testPipeline {
	return &testPipeline{name, ipVersion, NewTestPipeline(newSimulatedClock())}
}

func (p *testPipeline) fullname() string {
	return p.name + "." + p.ipVersion
}

func (p *testPipeline) testdataName(suffix string) string {
	return p.name + suffix
}

func (p *testPipeline) ipdataName(suffix string) string {
	return p.name + "." + p.ipVersion + suffix
}

func (p *testPipeline) handleCustomRequest(t *testing.T, method, mimeType string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, "https://example.com/upload/", bytes.NewReader(testdata(t, p.testdataName(".json"))))
	request.Header.Add("Content-Type", mimeType)
	if p.ipVersion == "ipv6" {
		request.RemoteAddr = "[2001:db8::2]:1234"
	}
	var response httptest.ResponseRecorder
	p.pipeline.ServeHTTP(&response, request)
	return &response
}

func (p *testPipeline) handleRequest(t *testing.T) bool {
	response := p.handleCustomRequest(t, "POST", "application/report")
	if response.Code != http.StatusNoContent {
		t.Errorf("ServeHTTP(%s): got %d, wanted %d", p.fullname(), p.ipVersion, response.Code, http.StatusNoContent)
		return false
	}
	return true
}

func allPipelineTests() []testPipeline {
	result := make([]testPipeline, len(testFiles)*2)
	for i := range testFiles {
		result[i*2] = *newTestPipeline(testFiles[i], "ipv4")
		result[i*2+1] = *newTestPipeline(testFiles[i], "ipv6")
	}
	return result
}

// Basic pipeline tests

func TestIgnoreNonPOST(t *testing.T) {
	pipeline := newTestPipeline("valid-nel-report", "")
	response := pipeline.handleCustomRequest(t, "GET", "application/report")
	if response.Code != http.StatusMethodNotAllowed {
		t.Errorf("ServeHTTP(%s): got %d, wanted %d", pipeline.fullname(), response.Code, http.StatusMethodNotAllowed)
		return
	}
}

func TestIgnoreWrongContentType(t *testing.T) {
	pipeline := newTestPipeline("valid-nel-report", "")
	response := pipeline.handleCustomRequest(t, "POST", "application/json")
	if response.Code != http.StatusBadRequest {
		t.Errorf("ServeHTTP(%s): got %d, wanted %d", pipeline.fullname(), response.Code, http.StatusBadRequest)
		return
	}
}

// CLF log dumping test cases

func TestDumpReports(t *testing.T) {
	for _, p := range allPipelineTests() {
		t.Run("Dump:"+p.fullname(), func(t *testing.T) {
			var buffer bytes.Buffer
			p.pipeline.AddProcessor(ReportDumper{&buffer})
			if !p.handleRequest(t) {
				return
			}

			got := buffer.Bytes()
			want := goldendata(t, p.ipdataName(".dumped.log"), got)
			if !cmp.Equal(got, want) {
				t.Errorf("ReportDumper(%s) == %s, wanted %s", p.fullname(), got, want)
				return
			}
		})
	}
}

// Custom annotations

var clientCountries = map[string]string{
	"192.0.2.1": "US",
	"192.0.2.2": "CA",
}

var serverZones = map[string]string{
	"203.0.113.75": "us-east1-a",
	"203.0.113.76": "us-west1-b",
}

type geoAnnotator struct{}

func (g geoAnnotator) ProcessReports(batch *ReportBatch) {
	batch.Annotation = clientCountries[batch.ClientIP]
	for i := range batch.Reports {
		batch.Reports[i].Annotation = serverZones[batch.Reports[i].ServerIP]
	}
}

type stashReports struct {
	dest *ReportBatch
}

func (s stashReports) ProcessReports(batch *ReportBatch) {
	*s.dest = *batch
}

func TestCustomAnnotation(t *testing.T) {
	for _, p := range allPipelineTests() {
		t.Run("Annotate:"+p.fullname(), func(t *testing.T) {
			var batch ReportBatch
			p.pipeline.AddProcessor(&geoAnnotator{})
			p.pipeline.AddProcessor(&stashReports{&batch})
			if !p.handleRequest(t) {
				return
			}

			got, err := encodeRawBatch(batch)
			if err != nil {
				t.Errorf("encodeRawBatch(%s): %v", p.fullname(), err)
				return
			}

			want := goldendata(t, p.ipdataName(".annotated.json"), got)
			if !cmp.Equal(got, want) {
				t.Errorf("ReportDumper(%s) == %s, wanted %s", p.fullname(), got, want)
				return
			}
		})
	}
}
