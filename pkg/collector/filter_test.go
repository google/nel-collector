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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nel-collector/pkg/collector"
	"github.com/google/nel-collector/pkg/pipelinetest"
)

func TestKeepNelReports(t *testing.T) {
	for _, p := range allPipelineTests() {
		t.Run("KeepNelReports:"+p.fullname(), func(t *testing.T) {
			var batch collector.ReportBatch
			p.AddProcessor(&collector.KeepNelReports{})
			p.AddProcessor(&pipelinetest.StashReports{&batch})
			err := p.HandleRequest(t, testdata(t, p.testdataName(".json")))
			if err != nil {
				t.Errorf("HandleRequest(%s): %v", p.fullname(), err)
				return
			}

			got, err := collector.EncodeRawBatch(batch)
			if err != nil {
				t.Errorf("EncodeRawBatch(%s): %v", p.fullname(), err)
				return
			}

			want := goldendata(t, p.ipdataName(".keep-nel.json"), got)
			if !cmp.Equal(got, want) {
				t.Errorf("ReportDumper(%s) == %s, wanted %s", p.fullname(), got, want)
				return
			}
		})
	}
}
