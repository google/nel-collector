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

type simulatedClock struct {
	currentTime time.Time
}

func newSimulatedClock() *simulatedClock {
	return &simulatedClock{currentTime: time.Unix(0, 0)}
}

func (c simulatedClock) Now() time.Time {
	return c.currentTime
}

var validNELReport = []byte(`[
		  {
		    "age": 500,
		    "type": "network-error",
		    "url": "https://example.com/about/",
		    "body": {
		      "uri": "https://example.com/about/",
		      "referrer": "https://example.com/",
		      "sampling-fraction": 0.5,
		      "server-ip": "203.0.113.75",
		      "protocol": "h2",
		      "status-code": 200,
		      "elapsed-time": 45,
		      "type": "ok"
		    }
		  }
		]`)

var nonNELReport = []byte(`[
		  {
		    "age": 500,
		    "type": "another-error",
		    "url": "https://example.com/about/",
		    "body": {"random": "stuff", "ignore": 100}
		  }
		]`)

func TestIgnoreNonPOST(t *testing.T) {
	pipeline := NewTestPipeline(newSimulatedClock())
	request := httptest.NewRequest("GET", "https://example.com/upload/", bytes.NewReader(validNELReport))
	request.Header.Add("Content-Type", "application/report")
	var response httptest.ResponseRecorder
	pipeline.ServeHTTP(&response, request)
	if response.Code != http.StatusMethodNotAllowed {
		t.Errorf("ServeHTTP(GET): got %d, wanted %d", response.Code, http.StatusMethodNotAllowed)
		return
	}
}

func TestIgnoreWrongContentType(t *testing.T) {
	pipeline := NewTestPipeline(newSimulatedClock())
	request := httptest.NewRequest("POST", "https://example.com/upload/", bytes.NewReader(validNELReport))
	request.Header.Add("Content-Type", "application/json")
	var response httptest.ResponseRecorder
	pipeline.ServeHTTP(&response, request)
	if response.Code != http.StatusBadRequest {
		t.Errorf("ServeHTTP(GET): got %d, wanted %d", response.Code, http.StatusBadRequest)
		return
	}
}

var dumpCases = []struct {
	name   string
	json   []byte
	dumped []byte
}{
	{
		"ValidNELReport",
		validNELReport,
		[]byte("1970-01-01 00:00:00.000 [ok] https://example.com/about/\n"),
	},
	{
		"NonNELReport",
		nonNELReport,
		[]byte("1970-01-01 00:00:00.000 <another-error> https://example.com/about/\n"),
	},
}

func TestDumpReports(t *testing.T) {
	for _, c := range dumpCases {
		t.Run("Dump:"+c.name, func(t *testing.T) {
			pipeline := NewTestPipeline(newSimulatedClock())
			var buffer bytes.Buffer
			pipeline.AddProcessor(ReportDumper{&buffer})

			request := httptest.NewRequest("POST", "https://example.com/upload/", bytes.NewReader(c.json))
			request.Header.Add("Content-Type", "application/report")
			var response httptest.ResponseRecorder
			pipeline.ServeHTTP(&response, request)

			if response.Code != http.StatusNoContent {
				t.Errorf("ServeHTTP(%s): got %d, wanted %d", compactJSON(c.json), response.Code, http.StatusNoContent)
				return
			}

			got := buffer.Bytes()
			if !cmp.Equal(got, c.dumped) {
				t.Errorf("ReportDumper(%s) == %s, wanted %s", compactJSON(c.json), got, c.dumped)
				return
			}
		})
	}
}
