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
	"encoding/json"
	"flag"
	"io/ioutil"
	"path/filepath"
	"testing"
)

var update = flag.Bool("update", false, "update .golden files")

// testFiles is a list of all of the test cases that have an input data file in
// the testdata/ subdirectory.  The list should be in the order that you want
// tests to be run; simpler or more basic test cases should appear first.
var testFiles = []string{
	"valid-nel-report",
	"multiple-valid-nel-reports",
	"non-nel-report",
}

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
