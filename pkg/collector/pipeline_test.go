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
	"flag"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/google/nel-collector/pkg/collector"
	"github.com/google/nel-collector/pkg/pipelinetest"
)

var update = flag.Bool("update", false, "update .golden files")

// Basic pipeline tests

var validNelReportPath = filepath.Clean("../pipelinetest/testdata/reports/valid-nel-report.json")

func TestIgnoreNonPOST(t *testing.T) {
	pipeline := collector.NewPipeline(pipelinetest.NewSimulatedClock())
	request := httptest.NewRequest("GET", "https://example.com/upload/", bytes.NewReader(testdata(validNelReportPath)))
	request.Header.Add("Content-Type", "application/report")
	var response httptest.ResponseRecorder
	pipeline.ServeHTTP(&response, request)
	if want := http.StatusMethodNotAllowed; response.Code != want {
		t.Errorf("ServeHTTP(method=GET): got %d, wanted %d", response.Code, want)
		return
	}
}

func TestIgnoreWrongContentType(t *testing.T) {
	pipeline := collector.NewPipeline(pipelinetest.NewSimulatedClock())
	request := httptest.NewRequest("POST", "https://example.com/upload/", bytes.NewReader(testdata(validNelReportPath)))
	request.Header.Add("Content-Type", "application/json")
	var response httptest.ResponseRecorder
	pipeline.ServeHTTP(&response, request)
	if want := http.StatusBadRequest; response.Code != want {
		t.Errorf("ServeHTTP(Content-Type=application/json): got %d, wanted %d", response.Code, want)
		return
	}
}

func TestProcessReports(t *testing.T) {
	pipeline := collector.NewPipeline(pipelinetest.NewSimulatedClock())
	pipeline.AddProcessor(pipelinetest.EncodeBatchAsResult{})
	p := pipelinetest.PipelineTest{
		TestName:          "TestProcessReports",
		Pipeline:          pipeline,
		InputPath:         "../pipelinetest",
		UpdateGoldenFiles: *update,
	}
	p.Run(t)
}

// CLF log dumping test cases

func TestDumpReports(t *testing.T) {
	pipeline := collector.NewPipeline(pipelinetest.NewSimulatedClock())
	pipeline.AddProcessor(collector.ReportDumper{})
	p := pipelinetest.PipelineTest{
		TestName:          "TestDumpReports",
		Pipeline:          pipeline,
		InputPath:         "../pipelinetest",
		OutputExtension:   ".log",
		UpdateGoldenFiles: *update,
	}
	p.Run(t)
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
	batch.SetAnnotation("ClientCountry", clientCountries[batch.ClientIP])
	for i := range batch.Reports {
		batch.Reports[i].SetAnnotation("ServerZone", serverZones[batch.Reports[i].ServerIP])
	}
}

func TestCustomAnnotation(t *testing.T) {
	pipeline := collector.NewPipeline(pipelinetest.NewSimulatedClock())
	pipeline.AddProcessor(&geoAnnotator{})
	pipeline.AddProcessor(pipelinetest.EncodeBatchAsResult{})
	p := pipelinetest.PipelineTest{
		TestName:          "TestCustomAnnotation",
		Pipeline:          pipeline,
		InputPath:         "../pipelinetest",
		UpdateGoldenFiles: *update,
	}
	p.Run(t)
}
