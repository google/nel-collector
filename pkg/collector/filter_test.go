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
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestKeepNelReports(t *testing.T) {
	for _, p := range allPipelineTests() {
		t.Run("KeepNelReports:"+p.fullname(), func(t *testing.T) {
			var batch ReportBatch
			p.pipeline.AddProcessor(&KeepNelReports{})
			p.pipeline.AddProcessor(&stashReports{&batch})
			if !p.handleRequest(t) {
				return
			}

			got, err := encodeRawBatch(batch)
			if err != nil {
				t.Errorf("encodeRawBatch(%s): %v", p.fullname(), err)
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
