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
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func compactJSON(b []byte) []byte {
	var bytes bytes.Buffer
	err := json.Compact(&bytes, b)
	if err != nil {
		return nil
	}
	return bytes.Bytes()
}

func TestNelReport(t *testing.T) {
	cases := []struct {
		json   []byte
		parsed []NelReport
	}{
		// A perfectly valid NEL report
		{
			[]byte(`[
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
			]`),
			[]NelReport{
				NelReport{
					Age:              500,
					ReportType:       "network-error",
					URL:              "https://example.com/about/",
					Referrer:         "https://example.com/",
					SamplingFraction: 0.5,
					ServerIP:         "203.0.113.75",
					Protocol:         "h2",
					StatusCode:       200,
					ElapsedTime:      45,
					Type:             "ok",
				},
			},
		},

		// We ignore the body if for non-NEL reports
		{
			[]byte(`[
			  {
			    "age": 500,
			    "type": "another-error",
			    "url": "https://example.com/about/",
			    "body": {"random": "stuff", "ignore": 100}
			  }
			]`),
			[]NelReport{
				NelReport{
					Age:        500,
					ReportType: "another-error",
					URL:        "https://example.com/about/",
					RawBody:    []byte(`{"random": "stuff", "ignore": 100}`),
				},
			},
		},
	}

	// First test unmarshaling
	for _, c := range cases {
		var got []NelReport
		err := json.Unmarshal(c.json, &got)
		if err != nil {
			t.Errorf("json.Unmarshal(%s): %v", compactJSON(c.json), err)
			continue
		}

		if !cmp.Equal(got, c.parsed) {
			t.Errorf("json.Unmarshal(%s) == %v, want %v", compactJSON(c.json), got, c.parsed)
		}
	}

	// Then test marshaling
	for _, c := range cases {
		got, err := json.Marshal(c.parsed)
		if err != nil {
			t.Errorf("json.Marshal(%v): %v", c.parsed, err)
			continue
		}

		if !cmp.Equal(compactJSON(got), compactJSON(c.json)) {
			t.Errorf("json.Marshal(%v) == %s, want %s", c.parsed, compactJSON(got), compactJSON(c.json))
		}
	}
}
