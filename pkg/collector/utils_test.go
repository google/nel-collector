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
	"io/ioutil"
	"path/filepath"
	"testing"
)

// testdata loads the contents of a file in the testdata/ subdirectory.
func testdata(t *testing.T, relPath string) []byte {
	fullPath := filepath.Join("testdata", relPath)
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
