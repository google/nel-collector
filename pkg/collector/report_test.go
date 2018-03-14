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
	"testing"

	"github.com/google/go-cmp/cmp"
)

var jsonCases = []struct {
	name     string
	jsonFile string
	parsed   []NelReport
}{
	{
		"ValidNELReport",
		"valid-nel-report.json",
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
	{
		"NonNELReport",
		"non-nel-report.json",
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

func TestNelReport(t *testing.T) {
	// First test unmarshaling
	for _, c := range jsonCases {
		t.Run("Unmarshal:"+c.name, func(t *testing.T) {
			jsonData := testdata(t, c.jsonFile)

			var got []NelReport
			err := json.Unmarshal(jsonData, &got)
			if err != nil {
				t.Errorf("json.Unmarshal(%s): %v", compactJSON(jsonData), err)
				return
			}

			if !cmp.Equal(got, c.parsed) {
				t.Errorf("json.Unmarshal(%s) == %v, want %v", compactJSON(jsonData), got, c.parsed)
			}
		})
	}

	// Then test marshaling
	for _, c := range jsonCases {
		t.Run("Marshal:"+c.name, func(t *testing.T) {
			want := testdata(t, c.jsonFile)

			got, err := json.Marshal(c.parsed)
			if err != nil {
				t.Errorf("json.Marshal(%v): %v", c.parsed, err)
				return
			}

			if !cmp.Equal(compactJSON(got), compactJSON(want)) {
				t.Errorf("json.Marshal(%v) == %s, want %s", c.parsed, compactJSON(got), compactJSON(want))
			}
		})
	}
}
