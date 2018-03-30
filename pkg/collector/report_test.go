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
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nel-collector/pkg/collector"
)

// JSON marshalling and unmarshalling

func TestNelReport(t *testing.T) {
	// Note: when updating golden files, we assume that the "unparsed" file is
	// canonical, and update the contents of the parsed file.

	// First test unmarshaling
	for _, name := range testFiles {
		t.Run("Unmarshal:"+name, func(t *testing.T) {
			jsonFile := filepath.Join("../pipelinetest/testdata/reports", name+".json")
			parsedFile := filepath.Join("testdata/TestNelReport", name+".json")
			jsonData := testdata(jsonFile)

			var reports []collector.NelReport
			err := json.Unmarshal(jsonData, &reports)
			if err != nil {
				t.Errorf("json.Unmarshal(%s): %v", name, err)
				return
			}

			got, err := collector.EncodeRawReports(reports)
			if err != nil {
				t.Errorf("EncodeRawReports(%s): %v", name, err)
				return
			}

			want := goldendata(parsedFile, got)
			if !cmp.Equal(compactJSON(got), compactJSON(want)) {
				t.Errorf("json.Unmarshal(%s) == %s, want %s", name, compactJSON(got), compactJSON(want))
			}
		})
	}

	// Then test marshaling
	for _, name := range testFiles {
		t.Run("Marshal:"+name, func(t *testing.T) {
			jsonFile := filepath.Join("../pipelinetest/testdata/reports", name+".json")
			parsedFile := filepath.Join("testdata/TestNelReport", name+".json")
			parsedData := testdata(parsedFile)

			var reports []collector.NelReport
			err := collector.DecodeRawReports(parsedData, &reports)
			if err != nil {
				t.Errorf("DecodeRawReports(%s): %v", name, err)
				return
			}

			got, err := json.Marshal(reports)
			if err != nil {
				t.Errorf("json.Marshal(%s): %v", name, err)
				return
			}

			want := testdata(jsonFile)
			if !cmp.Equal(compactJSON(got), compactJSON(want)) {
				t.Errorf("json.Marshal(%s) == %s, want %s", name, compactJSON(got), compactJSON(want))
			}
		})
	}
}
