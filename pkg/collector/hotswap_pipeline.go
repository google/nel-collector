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
	"net/http"
	"sync"
)

// A HotSwapPipeline implements the same interface as a normal Pipeline, but with the added support for a new Pipeline to be swapped out in the middle of execution. In this manner, an external listener can load a new configuration, create a new Pipeline, and call swap. In a threadsafe manner, the old Pipeline will be replaced and all future calls to ServeHTTP will use the new version of the pipeline.
type HotSwapPipeline struct {
	mux  sync.RWMutex
	pipe Pipeline
}

// Swap takes a new Pipeline and atomicly exchanges a new pipeline for all future processing.
func (p *HotSwapPipeline) Swap(newPipeline *Pipeline) {
	p.mux.Lock()
	defer p.mux.Unlock()
	p.pipe = *newPipeline
}

// getPipeline safely retrieves the current pipeline
func (p *HotSwapPipeline) getPipeline() *Pipeline {
	p.mux.RLock()
	defer p.mux.RUnlock()
	return &p.pipe
}

// ServeHTTP handles POST report uploads, extracting the payload and handing it
// off to ProcessReports for processing.
func (p *HotSwapPipeline) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pipeline := p.getPipeline()
	pipeline.ServeHTTP(w, r)
}
