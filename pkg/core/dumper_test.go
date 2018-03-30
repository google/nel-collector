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

package core_test

import (
	"testing"

	"github.com/google/nel-collector/pkg/collector"
	"github.com/google/nel-collector/pkg/core"
	"github.com/google/nel-collector/pkg/pipelinetest"
)

// CLF log dumping test cases

func TestDumpReportsAsCLF(t *testing.T) {
	pipeline := collector.NewPipeline(pipelinetest.NewSimulatedClock())
	pipeline.AddProcessor(core.DumpReportsAsCLF{})
	p := pipelinetest.PipelineTest{
		TestName:          "TestDumpReportsAsCLF",
		Pipeline:          pipeline,
		InputPath:         "../pipelinetest",
		OutputExtension:   ".log",
		UpdateGoldenFiles: *update,
	}
	p.Run(t)
}
