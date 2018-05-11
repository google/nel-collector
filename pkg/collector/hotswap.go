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

// A HotSwap wraps a http.Handler and adds the support for a new handler to
// swapped in the middle of execution with no interruption to processing. In
// this manner, an external listener can load a new configuration, create a new
// Handler, and call Swap. In a threadsafe manner, the old Handler will be
// replaced and all future calls to ServeHTTP will use the new version of the
// handler.
type HotSwap struct {
	mux     sync.RWMutex
	handler http.Handler
}

// Swap takes a new Pipeline and atomicly exchanges a new handler for all
// future processing.
func (h *HotSwap) Swap(newHandler http.Handler) {
	h.mux.Lock()
	defer h.mux.Unlock()
	h.handler = newHandler
}

// getHandler safely retrieves the current handler.
func (h *HotSwap) getHandler() http.Handler {
	h.mux.RLock()
	defer h.mux.RUnlock()
	return h.handler
}

// ServeHTTP delegates incoming requests to the contained handler in a thread
// safe manner.
func (h *HotSwap) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handler := h.getHandler()
	handler.ServeHTTP(w, r)
}
