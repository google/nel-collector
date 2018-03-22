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

package collector_test

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nel-collector/pkg/collector"
	"github.com/google/nel-collector/pkg/pipelinetest"
)

// Helpers

type testCase struct {
	name      string
	ipVersion string
	*pipelinetest.TestPipeline
}

func newTestCase(name, ipVersion string) *testCase {
	var remoteAddr string
	if ipVersion == "ipv6" {
		remoteAddr = "[2001:db8::2]:1234"
	}
	return &testCase{name, ipVersion, pipelinetest.NewTestPipeline(remoteAddr)}
}

func (p *testCase) fullname() string {
	return p.name + "." + p.ipVersion
}

func (p *testCase) testdataName(suffix string) string {
	return p.name + suffix
}

func (p *testCase) ipdataName(suffix string) string {
	return p.name + "." + p.ipVersion + suffix
}

func allPipelineTests() []testCase {
	result := make([]testCase, len(testFiles)*2)
	for i := range testFiles {
		result[i*2] = *newTestCase(testFiles[i], "ipv4")
		result[i*2+1] = *newTestCase(testFiles[i], "ipv6")
	}
	return result
}

// Basic pipeline tests

func TestIgnoreNonPOST(t *testing.T) {
	pipeline := newTestCase("valid-nel-report", "")
	response := pipeline.HandleCustomRequest(t, "GET", "application/report", testdata(t, pipeline.testdataName(".json")))
	if response.Code != http.StatusMethodNotAllowed {
		t.Errorf("ServeHTTP(%s): got %d, wanted %d", pipeline.fullname(), response.Code, http.StatusMethodNotAllowed)
		return
	}
}

func TestIgnoreWrongContentType(t *testing.T) {
	pipeline := newTestCase("valid-nel-report", "")
	response := pipeline.HandleCustomRequest(t, "POST", "application/json", testdata(t, pipeline.testdataName(".json")))
	if response.Code != http.StatusBadRequest {
		t.Errorf("ServeHTTP(%s): got %d, wanted %d", pipeline.fullname(), response.Code, http.StatusBadRequest)
		return
	}
}

type stashReports struct {
	dest *collector.ReportBatch
}

func (s stashReports) ProcessReports(batch *collector.ReportBatch) {
	*s.dest = *batch
}

func TestProcessReports(t *testing.T) {
	for _, p := range allPipelineTests() {
		t.Run("Process:"+p.fullname(), func(t *testing.T) {
			var batch collector.ReportBatch
			p.AddProcessor(&stashReports{&batch})
			err := p.HandleRequest(t, testdata(t, p.testdataName(".json")))
			if err != nil {
				t.Errorf("HandleRequest(%s): %v", p.fullname(), err)
				return
			}

			got, err := encodeRawBatch(batch)
			if err != nil {
				t.Errorf("encodeRawBatch(%s): %v", p.fullname(), err)
				return
			}

			want := goldendata(t, p.ipdataName(".processed.json"), got)
			if !cmp.Equal(got, want) {
				t.Errorf("ReportDumper(%s) == %s, wanted %s", p.fullname(), got, want)
				return
			}
		})
	}
}

// CLF log dumping test cases

func TestDumpReports(t *testing.T) {
	for _, p := range allPipelineTests() {
		t.Run("Dump:"+p.fullname(), func(t *testing.T) {
			var buffer bytes.Buffer
			p.AddProcessor(collector.ReportDumper{&buffer})
			err := p.HandleRequest(t, testdata(t, p.testdataName(".json")))
			if err != nil {
				t.Errorf("HandleRequest(%s): %v", p.fullname(), err)
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

func (g geoAnnotator) ProcessReports(batch *collector.ReportBatch) {
	batch.Annotation = clientCountries[batch.ClientIP]
	for i := range batch.Reports {
		batch.Reports[i].Annotation = serverZones[batch.Reports[i].ServerIP]
	}
}

func TestCustomAnnotation(t *testing.T) {
	for _, p := range allPipelineTests() {
		t.Run("Annotate:"+p.fullname(), func(t *testing.T) {
			var batch collector.ReportBatch
			p.AddProcessor(&geoAnnotator{})
			p.AddProcessor(&stashReports{&batch})
			err := p.HandleRequest(t, testdata(t, p.testdataName(".json")))
			if err != nil {
				t.Errorf("HandleRequest(%s): %v", p.fullname(), err)
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
