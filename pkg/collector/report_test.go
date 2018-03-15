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

var jsonCases = []struct{ name string }{
	{"valid-nel-report"},
	{"non-nel-report"},
}

func TestNelReport(t *testing.T) {
	// Note: when updating golden files, we assume that the "unparsed" file is
	// canonical, and update the contents of the parsed file.

	// This type alias lets us override our spec-aware JSON parsing rules, and
	// dump out the content of a NelReport instance exactly as it looks in Go.
	type ParsedNelReport NelReport

	// First test unmarshaling
	for _, c := range jsonCases {
		t.Run("Unmarshal:"+c.name, func(t *testing.T) {
			jsonFile := c.name + ".json"
			parsedFile := c.name + ".parsed.json"
			jsonData := testdata(t, jsonFile)

			var reports []NelReport
			err := json.Unmarshal(jsonData, &reports)
			if err != nil {
				t.Errorf("json.Unmarshal(%s): %v", c.name, err)
				return
			}

			parsedReports := make([]ParsedNelReport, len(reports))
			for i, _ := range reports {
				parsedReports[i] = (ParsedNelReport)(reports[i])
			}

			got, err := json.MarshalIndent(parsedReports, "", "  ")
			if err != nil {
				t.Errorf("json.Marshal(%s [parsed]): %v", c.name, err)
				return
			}

			want := goldendata(t, parsedFile, got)
			if !cmp.Equal(compactJSON(got), compactJSON(want)) {
				t.Errorf("json.Unmarshal(%s) == %v, want %v", c.name, compactJSON(got), compactJSON(want))
			}
		})
	}

	// Then test marshaling
	for _, c := range jsonCases {
		t.Run("Marshal:"+c.name, func(t *testing.T) {
			jsonFile := c.name + ".json"
			parsedFile := c.name + ".parsed.json"
			parsedData := testdata(t, parsedFile)

			var parsedReports []ParsedNelReport
			err := json.Unmarshal(parsedData, &parsedReports)
			if err != nil {
				t.Errorf("json.Unmarshal(%s [parsed]): %v", c.name, err)
				return
			}

			reports := make([]NelReport, len(parsedReports))
			for i, _ := range parsedReports {
				reports[i] = (NelReport)(parsedReports[i])
			}

			got, err := json.Marshal(reports)
			if err != nil {
				t.Errorf("json.Marshal(%s): %v", c.name, err)
				return
			}

			want := testdata(t, jsonFile)
			if !cmp.Equal(compactJSON(got), compactJSON(want)) {
				t.Errorf("json.Marshal(%s) == %v, want %v", c.name, compactJSON(got), compactJSON(want))
			}
		})
	}
}
