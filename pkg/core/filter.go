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
	"context"

	"github.com/BurntSushi/toml"
	"github.com/google/nel-collector/pkg/collector"
)

// KeepNelReports is a pipeline processor that throws away any non-NEL reports.
type KeepNelReports struct{}

// ProcessReports throws away any non-NEL reports.
func (KeepNelReports) ProcessReports(ctx context.Context, batch *collector.ReportBatch) {
	var filtered []collector.NelReport
	for _, report := range batch.Reports {
		if report.ReportType == "network-error" {
			filtered = append(filtered, report)
		}
	}
	batch.Reports = filtered
}

func init() {
	collector.RegisterReportLoaderFunc(
		"KeepNelReports",
		func(configPrimitive toml.Primitive) (collector.ReportProcessor, error) {
			return KeepNelReports{}, nil
		})
}
