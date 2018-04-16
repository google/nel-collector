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
	"fmt"

	"github.com/BurntSushi/toml"
)

// LoadFromConfig loads pipeline processors based on the contents of a TOML
// configuration file, and adds them to the pipeline.
//
// The configuration must have sections named `processor`, each of which defines
// one processor that should be added to the pipeline.  For instance, this
// configuration could look like:
//
//     [[processor]]
//     type = "KeepNelReports"
//
//     [[processor]]
//     type = "DumpReportsAsCLF"
//     use_utc = true
//
// The `type` field of each element identifies which kind of processor to add;
// any additional fields let you specify any processor-specific configuration.
func (p *Pipeline) LoadFromConfig(configBytes []byte) error {

	var config struct {
		Processors []toml.Primitive `toml:"processor"`
	}
	err := toml.Unmarshal(configBytes, &config)
	if err != nil {
		return fmt.Errorf("Invalid NEL configuration")
	}

	if config.Processors == nil {
		return fmt.Errorf("NEL configuration missing `processors`")
	}

	if len(config.Processors) == 0 {
		return fmt.Errorf("NEL configuration `processors` array must be non-empty")
	}

	for idx, processorPrimitive := range config.Processors {
		var processorConfig struct {
			Type string `toml:"type"`
		}
		err := toml.PrimitiveDecode(processorPrimitive, &processorConfig)
		if err != nil {
			return fmt.Errorf("Processor config 0 must be an object")
		}
		if processorConfig.Type == "" {
			return fmt.Errorf("Processor config %d is missing `type`", idx)
		}

		loader, ok := reportLoaders[processorConfig.Type]
		if !ok {
			return fmt.Errorf("Unknown processor type %s for processor %d", processorConfig.Type, idx)
		}

		processor, err := loader.Load(processorPrimitive)
		if err != nil {
			return fmt.Errorf("Couldn't create a %s for processor %d: %v", processorConfig.Type, idx, err)
		}

		p.AddProcessor(processor)
	}

	return nil
}

// ReportLoader is an interface that knows how to load a ReportProcessor at
// runtime via the contents of a TOML configuration file.
type ReportLoader interface {
	Load(config toml.Primitive) (ReportProcessor, error)
}

// ReportLoaderFunc allows you to register a simple function as a ReportLoader.
type ReportLoaderFunc func(config toml.Primitive) (ReportProcessor, error)

// Load defers to a ReportLoaderFunc to load a ReportProcessor from the contents
// of a configuration file.
func (f ReportLoaderFunc) Load(config toml.Primitive) (ReportProcessor, error) {
	return f(config)
}

var reportLoaders = make(map[string]ReportLoader)

// RegisterReportLoader registers a ReportLoader for a particular kind of report
// processor.
func RegisterReportLoader(name string, loader ReportLoader) {
	reportLoaders[name] = loader
}

// RegisterReportLoader registers a function that can load a particular kind of
// report processor.
func RegisterReportLoaderFunc(name string, loader func(config toml.Primitive) (ReportProcessor, error)) {
	RegisterReportLoader(name, ReportLoaderFunc(loader))
}
