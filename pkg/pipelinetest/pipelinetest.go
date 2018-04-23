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

// Package pipelinetest contains helper utilities for constructing test cases
// that exercise the components of a NEL collector pipeline.
package pipelinetest

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/google/nel-collector/pkg/collector"
	"github.com/kylelemons/godebug/diff"
)

// SimulatedClock is a Clock that gives you full control over which times are
// reported.  This can be used in test cases to give reproducible timestamps in
// your expected output.
type SimulatedClock struct {
	CurrentTime time.Time
}

// NewSimulatedClock returns a new SimulatedClock last initially reports the
// Unix expoch (midnight January 1, 1970) as the current time.
func NewSimulatedClock() *SimulatedClock {
	return &SimulatedClock{CurrentTime: time.Unix(0, 0).UTC()}
}

// Now returns the current time according to this SimulatedClock.
func (c SimulatedClock) Now() time.Time {
	return c.CurrentTime
}

// NewTestConfigPipeline constructs a new Pipeline from the contents of a TOML
// configuration string.  We assume that configString is well-formed and panic
// if there are any errors parsing it, or configuring the pipeline's processors.
// This is especially useful in test cases, along with PipelineTest.
func NewTestConfigPipeline(configString string) *collector.Pipeline {
	p := collector.NewPipeline(NewSimulatedClock())
	err := p.LoadFromConfig([]byte(configString))
	if err != nil {
		log.Fatal(err)
		return nil
	}
	return p
}

// PipelineTest automates the process of running a NEL collector pipeline
// against a large number of test uploads.
//
// We use testdata files to store several input report payloads, and golden
// files to hold the expected output of your pipeline for each of those
// payloads.  Running the test cases with the `--update` flag will overwrite the
// golden files with the current output from your pipeline.
//
// We expect the following directory structure:
//
//   [InputPath]/
//     testdata/
//       reports/
//         [payload-name].json
//   [OutputPath]/
//     testdata/
//       [TestName]/
//         [payload-name].ipv{4,6}.[OutputExtension]
//
// InputPath and OutputPath both default to the current directory if empty,
// which lines up with the `go test` convention of running test cases in the
// directory of the package being tested.
type PipelineTest struct {
	// The name of the test case that will use this helper.  Must be unique across
	// all test cases that use the same OutputPath.
	TestName string

	// The pipeline being tested.  It should include a processor that adds a
	// []byte annotation named `TestResult` to the report batch; we'll verify the
	// contents of this annotation to determine whether each test succeeded.
	Pipeline *collector.Pipeline

	// The path containing the testdata directory where we can find the files
	// containing the input report payloads.  For tests of package pipelinetest
	// itself, the default value ("") is correct.  If you want to test processors
	// in other packages, and reuse the input files from pipelinetest, set this to
	// the directory containing pipelinetest's testdata directory.
	InputPath string

	// The path containing the testdata directory where we can find the golden
	// files containing the expected output for your test case.  For most
	// packages, the default value ("") is correct.
	OutputPath string

	// The extension that we should use for the golden files for your test case.
	// If empty, we will use ".json".
	OutputExtension string

	// Whether to update the content of the golden files with the current actual
	// test output.  You'll usually set this to the value of an `--update`
	// command-line flag.
	UpdateGoldenFiles bool
}

// payloadNames returns all of the base filenames (not including the ".json"
// extension) of any input files found in the testdata directory.
func (p *PipelineTest) payloadNames() []string {
	var result []string
	basePath := filepath.Join(p.InputPath, "testdata", "reports")
	err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && path != basePath {
			return filepath.SkipDir
		}
		base := filepath.Base(path)
		if filepath.Ext(base) != ".json" {
			return nil
		}
		result = append(result, strings.TrimSuffix(base, ".json"))
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	return result
}

// inputPayload loads the contents of an input report payload.
func (p *PipelineTest) inputPayload(payloadName string) []byte {
	path := filepath.Join(p.InputPath, "testdata", "reports", payloadName+".json")
	content, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	return content
}

// expectedOutput loads the contents of a goldendata output file.  If the
// `--update` flags is set, we first update the file's contents with `got` (and
// will therefore always return `got`).
func (p *PipelineTest) expectedOutput(payloadName, ipTag string, got []byte) []byte {
	outputExtension := p.OutputExtension
	if outputExtension == "" {
		outputExtension = ".json"
	}
	path := filepath.Join(p.OutputPath, "testdata", p.TestName, payloadName+"."+ipTag+outputExtension)
	if p.UpdateGoldenFiles && got != nil {
		os.MkdirAll(filepath.Dir(path), 0755)
		ioutil.WriteFile(path, got, 0644)
	}
	content, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	return content
}

// Run tests your pipeline against all of the input files that we found in your
// InputPath, comparing the values of the TestResult annotation with the
// corresponding golden files in OutputPath.
func (p *PipelineTest) Run(t *testing.T) {
	for _, payloadName := range p.payloadNames() {
		for _, ip := range []struct{ tag, remoteAddr string }{{"ipv4", ""}, {"ipv6", "[2001:db8::2]:1234"}} {
			t.Run(p.TestName+":"+payloadName+":"+ip.tag, func(t *testing.T) {
				payload := p.inputPayload(payloadName)
				request := httptest.NewRequest("POST", "https://example.com/upload/", bytes.NewReader(payload))
				request.Header.Add("Content-Type", "application/report")
				if ip.remoteAddr != "" {
					request.RemoteAddr = ip.remoteAddr
				}

				var response httptest.ResponseRecorder
				batch := p.Pipeline.ProcessReports(&response, request)
				if response.Code != http.StatusNoContent {
					t.Errorf("ProcessReports(%s:%s) got status code %d, wanted %d", payloadName, ip.tag, response.Code, http.StatusNoContent)
					return
				}
				if batch == nil {
					t.Errorf("ProcessReports(%s:%s) got nil", payloadName, ip.tag)
					return
				}

				result := batch.GetAnnotation("TestResult")
				if result == nil {
					t.Errorf("TestResult(%s:%s) got nil", payloadName, ip.tag)
					return
				}

				got, ok := result.([]byte)
				if !ok {
					t.Errorf("TestResult(%s:%s) got %v, wanted []byte", payloadName, ip.tag, result)
				}

				want := p.expectedOutput(payloadName, ip.tag, got)
				if diff := diff.Diff((string)(want), (string)(got)); diff != "" {
					t.Errorf("TestResult(%s:%s) got diff (want → got):\n%s", payloadName, ip.tag, diff)
					return
				}
			})
		}
	}
}

// EncodeBatchAsResult is a pipeline processor that saves a copy of the report
// batch into the TestResult annotation.  You can use this with PipelineTest to
// use the full contents of the batch (including annotations) as the output to
// compare in your test case.
type EncodeBatchAsResult struct{}

// ProcessReports saves a copy of the report batch into the TestResult
// annotation.
func (e EncodeBatchAsResult) ProcessReports(batch *collector.ReportBatch) {
	encoded, _ := collector.EncodeRawBatch(batch)
	batch.SetAnnotation("TestResult", encoded)
}

func init() {
	collector.RegisterReportLoaderFunc(
		"EncodeBatchAsResult",
		func(configPrimitive toml.Primitive) (collector.ReportProcessor, error) {
			return EncodeBatchAsResult{}, nil
		})
}
