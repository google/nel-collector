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

package core

import (
	"io"

	"github.com/google/nel-collector/pkg/collector"
)

// DumpReportsAsCLF is a ReportProcessor that prints out a summary of each
// report using a format not unlike Apache's CLF access.log format.
type DumpReportsAsCLF struct {
	// Writer is where the report summaries should be written to.  If nil, we'll
	// save the summaries as the value of the TestResult annotation.
	Writer io.Writer
}

// ProcessReports prints out a summary of each report in the batch.
func (d DumpReportsAsCLF) ProcessReports(batch *collector.ReportBatch) {
	writer := d.Writer
	if writer == nil {
		writer = batch.AnnotationWriter("TestResult")
	}
	collector.PrintBatchAsCLF(batch, writer)
}
