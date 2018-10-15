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
	"context"
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
type PipelineTest struct {
	// The name of the test case that will use this helper.  Must be unique across
	// all test cases that use the same OutputPath.
	TestName string

	// The pipeline being tested.  It should include a processor that adds a
	// []byte annotation named `TestResult` to the report batch; we'll verify the
	// contents of this annotation to determine whether each test succeeded.
	Pipeline *collector.Pipeline

	// The extension that we should use for the golden files for your test case.
	// If empty, we will use ".json".
	OutputExtension string

	// The loader that will be used to read input and output files for each test
	// case.
	Testdata TestdataLoader
}

// TestCase describes one test case managed by a PipelineTest.
type TestCase struct {
	// The name of the PipelineTest that created this test case.
	TestName string

	// The name of the payload file that is used as input.  This will be the base
	// filename of one of the files in the PipelineTest's InputPath, with the
	// .json extension removed.
	PayloadName string

	// Whether the payload is fake-uploaded via `ipv4` or `ipv6`.
	IPTag string

	// The golden file extension for this test case.  Will never be empty.
	OutputExtension string
}

// BaseInputFilename returns the base filename of the input file for a test
// case.
func (c TestCase) BaseInputFilename() string {
	return c.PayloadName + ".json"
}

// BaseOutputFilename returns the base filename of the output file for a test
// case.
func (c TestCase) BaseOutputFilename() string {
	return c.PayloadName + "." + c.IPTag + c.OutputExtension
}

// Name returns the name of the test case, relative to the test name.
func (c TestCase) Name() string {
	return c.PayloadName + ":" + c.IPTag
}

// FullName returns the full name of the test case, including the overall test
// name.
func (c TestCase) FullName() string {
	return c.TestName + "/" + c.Name()
}

// Run tests your pipeline against all of the input files that we found in your
// InputPath, comparing the values of the TestResult annotation with the
// corresponding golden files in OutputPath.
func (p *PipelineTest) Run(t *testing.T) {
	payloadNames, err := p.Testdata.GetPayloadNames()
	if err != nil {
		t.Fatal(err)
	}

	outputExtension := p.OutputExtension
	if outputExtension == "" {
		outputExtension = ".json"
	}

	for _, payloadName := range payloadNames {
		for _, ip := range []struct{ tag, remoteAddr string }{{"ipv4", ""}, {"ipv6", "[2001:db8::2]:1234"}} {
			testCase := TestCase{p.TestName, payloadName, ip.tag, outputExtension}
			t.Run(testCase.Name(), func(t *testing.T) {
				payload, err := p.Testdata.LoadInputFile(testCase)
				if err != nil {
					t.Fatal(err)
					return
				}

				request := httptest.NewRequest("POST", "https://example.com/upload/", bytes.NewReader(payload))
				request.Header.Add("Content-Type", "application/report")
				if ip.remoteAddr != "" {
					request.RemoteAddr = ip.remoteAddr
				}

				var response httptest.ResponseRecorder
				batch := p.Pipeline.ProcessReports(context.Background(), &response, request)
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

				want, err := p.Testdata.LoadOutputFile(testCase, got)
				if err != nil {
					t.Fatal(err)
				}
				if diff := diff.Diff((string)(want), (string)(got)); diff != "" {
					t.Errorf("TestResult(%s:%s) got diff (want → got):\n%s", payloadName, ip.tag, diff)
					return
				}
			})
		}
	}
}

// TestdataLoader is a helper interface that PipelineTest uses to find, read,
// and write the testdata and golden files for a set of test cases.
type TestdataLoader interface {
	// GetPayloadNames finds all available input files and returns their
	// PayloadNames.
	GetPayloadNames() ([]string, error)

	// LoadInputFile loads in the content of the input file for a particular test
	// case.
	LoadInputFile(testCase TestCase) ([]byte, error)

	// LoadOutputFile loads in the content of the golden output file for a
	// particular test case.  If the test is run in "update" mode (which is up to
	// you to decide, typically via an `--update` flag), then you should replace
	// any existing content with `got`, and then return `got`.
	LoadOutputFile(testCase TestCase, got []byte) ([]byte, error)
}

// DefaultTestdataLoader looks for test and golden data files in `testdata`
// directories in the source packages being tested.
//
// We expect the following directory structure:
//
//   [InputPath]/
//     testdata/
//       reports/
//         [PayloadName].json
//   [OutputPath]/
//     testdata/
//       [TestName]/
//         [PayloadName].[IPTag].[OutputExtension]
//
// InputPath and OutputPath both default to the current directory if empty,
// which lines up with the `go test` convention of running test cases in the
// directory of the package being tested.
type DefaultTestdataLoader struct {
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

	// Whether to update the content of the golden files with the current actual
	// test output.  You'll usually set this to the value of an `--update`
	// command-line flag.
	UpdateGoldenFiles bool
}

// GetPayloadNames returns the PayloadNames of any input files found in a
// particular directory.  (This makes it easy to write TestdataLoader instances
// that load input files from the local filesystem; all you have to do in your
// GetPayloadNames method is identify the correct path, and then delegate to
// this helper function.)
func GetPayloadNames(basePath string) ([]string, error) {
	var result []string
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
		return nil, err
	}
	return result, nil
}

// GetPayloadNames returns the PayloadNames of any input files found in
// InputPath.
func (l DefaultTestdataLoader) GetPayloadNames() ([]string, error) {
	return GetPayloadNames(filepath.Join(l.InputPath, "testdata", "reports"))
}

// LoadInputFile loads the contents of an input file from InputPath.
func (l DefaultTestdataLoader) LoadInputFile(testCase TestCase) ([]byte, error) {
	path := filepath.Join(l.InputPath, "testdata", "reports", testCase.BaseInputFilename())
	return ioutil.ReadFile(path)
}

// LoadOutputFile loads the contents of a golden file from OutputPath, updating
// its contents with `got` if UpdateGoldenFiles is true.
func (l DefaultTestdataLoader) LoadOutputFile(testCase TestCase, got []byte) ([]byte, error) {
	path := filepath.Join(l.OutputPath, "testdata", testCase.TestName, testCase.BaseOutputFilename())
	if l.UpdateGoldenFiles && got != nil {
		err := os.MkdirAll(filepath.Dir(path), 0755)
		if err != nil {
			return nil, err
		}
		err = ioutil.WriteFile(path, got, 0644)
		if err != nil {
			return nil, err
		}
	}
	return ioutil.ReadFile(path)
}

// EncodeBatchAsResult is a pipeline processor that saves a copy of the report
// batch into the TestResult annotation.  You can use this with PipelineTest to
// use the full contents of the batch (including annotations) as the output to
// compare in your test case.
type EncodeBatchAsResult struct{}

// ProcessReports saves a copy of the report batch into the TestResult
// annotation.
func (e EncodeBatchAsResult) ProcessReports(ctx context.Context, batch *collector.ReportBatch) {
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
