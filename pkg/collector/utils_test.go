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
	"flag"
	"io/ioutil"
	"path/filepath"
	"testing"
)

var update = flag.Bool("update", false, "update .golden files")

// testdata loads the contents of a file in the testdata/ subdirectory.
func testdata(t *testing.T, relPath string) []byte {
	fullPath := filepath.Join("testdata", relPath)
	content, err := ioutil.ReadFile(fullPath)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

// goldendata loads the contents of a file in the testdata/ subdirectory,
// updating the contents of the file first (with `got`) if the --update flag is
// given.
func goldendata(t *testing.T, relPath string, got []byte) []byte {
	fullPath := filepath.Join("testdata", relPath)
	if *update {
		ioutil.WriteFile(fullPath, got, 0644)
	}
	content, err := ioutil.ReadFile(fullPath)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

// compactJSON compacts a JSON blob so that it (among other things) fits onto
// one line.
func compactJSON(b []byte) []byte {
	var bytes bytes.Buffer
	err := json.Compact(&bytes, b)
	if err != nil {
		return nil
	}
	return bytes.Bytes()
}

// encodeRawReports marshals an array of NelReports without using our custom
// spec-aware JSON parsing rules, instead dumping out the content exactly as it
// looks in Go.  This is used extensively in test cases to compare the results
// of parsing and annotating against golden files.
func encodeRawReports(reports []NelReport) ([]byte, error) {
	// This type alias lets us override our spec-aware JSON parsing rules, and
	// dump out the content of a NelReport instance exactly as it looks in Go.
	type ParsedNelReport NelReport
	parsedReports := make([]ParsedNelReport, len(reports))
	for i := range reports {
		parsedReports[i] = (ParsedNelReport)(reports[i])
	}
	return json.MarshalIndent(parsedReports, "", "  ")
}

// decodeRawReports unmarshals an array of NelReports without using our custom
// spec-aware JSON parsing rules.  It's the inverse of encodeRawReports.
func decodeRawReports(b []byte, reports *[]NelReport) error {
	// This type alias lets us override our spec-aware JSON parsing rules, and
	// dump out the content of a NelReport instance exactly as it looks in Go.
	type ParsedNelReport NelReport
	var parsedReports []ParsedNelReport
	err := json.Unmarshal(b, &parsedReports)
	if err != nil {
		return err
	}

	*reports = make([]NelReport, len(parsedReports))
	for i := range parsedReports {
		(*reports)[i] = (NelReport)(parsedReports[i])
	}
	return nil
}

// encodeRawBatch marshals a batch of NelReports, including any custom
// annotations, without using our custom spec-aware JSON parsing rules.
func encodeRawBatch(batch ReportBatch) ([]byte, error) {
	var err error
	var rawBatch struct {
		ReportBatch
		RawReports json.RawMessage `json:"Reports"`
	}

	rawBatch.ReportBatch = batch
	rawBatch.RawReports, err = encodeRawReports(rawBatch.Reports)
	if err != nil {
		return nil, err
	}

	rawBatch.Reports = nil
	return json.MarshalIndent(rawBatch, "", "  ")
}
