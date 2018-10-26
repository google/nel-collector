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
	"context"
	"fmt"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/google/nel-collector/pkg/collector"
	"github.com/kylelemons/godebug/diff"
)

var badConfigCases = []struct {
	name, configYaml, expectedError string
}{
	{"EmptyConfig", ``,
		"NEL configuration missing `processors`"},
	{"ProcessorWrongType", `processor = 5`,
		"Invalid NEL configuration"},
	{"EmptyProcessors", `processor = []`,
		"NEL configuration `processors` array must be non-empty"},
	{"ProcessorWrongArrayType", `processor = [5]`,
		"Processor config 0 must be an object"},
	{"SecondProcessorWrongType", `processor = [{type = "UnknownType"}, 5]`,
		"Invalid NEL configuration"},
	{"ProcessorMissingType", `processor = [{}]`,
		"Processor config 0 is missing `type`"},
	{"UnknownProcessorType", `processor = [{type = "UnknownType"}]`,
		"Unknown processor type UnknownType for processor 0"},
	{"ErrorLoadingProcessor", `processor = [{type = "AlwaysThrowsError"}]`,
		"Couldn't create a AlwaysThrowsError for processor 0: this will never work"},
	{"ErrorLoadingContextProcessor", `processor = [{type = "AlwaysThrowsErrorWithContext"}]`,
		"Couldn't create a AlwaysThrowsErrorWithContext for processor 0: this will never work"},
}

func TestBadConfig(t *testing.T) {
	// Register a known processor type that always throws an error
	collector.RegisterReportLoaderFunc("AlwaysThrowsError", func(config toml.Primitive) (collector.ReportProcessor, error) {
		return nil, fmt.Errorf("this will never work")
	})
	// And another one that always throws an error while taking in a Context
	// parameter (even though it doesn't do anything with it.)
	collector.RegisterContextReportLoaderFunc("AlwaysThrowsErrorWithContext", func(_ context.Context, config toml.Primitive) (collector.ReportProcessor, error) {
		return nil, fmt.Errorf("this will never work")
	})
	for _, c := range badConfigCases {
		t.Run("BadConfig:"+c.name, func(t *testing.T) {
			var pipeline collector.Pipeline
			err := pipeline.LoadFromConfig(context.Background(), []byte(c.configYaml))
			if err == nil {
				t.Errorf("LoadFromConfig(%v) should return error", c.configYaml)
			}
			if diff := diff.Diff(c.expectedError, err.Error()); diff != "" {
				t.Errorf("LoadFromConfig(%v) got diff (want → got):\n%s", c.configYaml, diff)
			}
		})
	}
}
