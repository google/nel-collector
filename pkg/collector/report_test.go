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

// JSON marshalling and unmarshalling

func TestNelReport(t *testing.T) {
	// Note: when updating golden files, we assume that the "unparsed" file is
	// canonical, and update the contents of the parsed file.

	// First test unmarshaling
	for _, name := range testFiles {
		t.Run("Unmarshal:"+name, func(t *testing.T) {
			jsonFile := name + ".json"
			parsedFile := name + ".parsed.json"
			jsonData := testdata(t, jsonFile)

			var reports []NelReport
			err := json.Unmarshal(jsonData, &reports)
			if err != nil {
				t.Errorf("json.Unmarshal(%s): %v", name, err)
				return
			}

			got, err := encodeRawReports(reports)
			if err != nil {
				t.Errorf("encodeRawReports(%s): %v", name, err)
				return
			}

			want := goldendata(t, parsedFile, got)
			if !cmp.Equal(compactJSON(got), compactJSON(want)) {
				t.Errorf("json.Unmarshal(%s) == %s, want %s", name, compactJSON(got), compactJSON(want))
			}
		})
	}

	// Then test marshaling
	for _, name := range testFiles {
		t.Run("Marshal:"+name, func(t *testing.T) {
			jsonFile := name + ".json"
			parsedFile := name + ".parsed.json"
			parsedData := testdata(t, parsedFile)

			var reports []NelReport
			err := decodeRawReports(parsedData, &reports)
			if err != nil {
				t.Errorf("decodeRawReports(%s): %v", name, err)
				return
			}

			got, err := json.Marshal(reports)
			if err != nil {
				t.Errorf("json.Marshal(%s): %v", name, err)
				return
			}

			want := testdata(t, jsonFile)
			if !cmp.Equal(compactJSON(got), compactJSON(want)) {
				t.Errorf("json.Marshal(%s) == %s, want %s", name, compactJSON(got), compactJSON(want))
			}
		})
	}
}
