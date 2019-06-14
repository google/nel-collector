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

// HandlerCloser is an interface for a http.Handler that processes data
// asynchronously and therefore must be closed. Once Close is called,
// the caller must ensure that no further calls to ServeHTTP are made.
type HandlerCloser interface {
	Close()
	http.Handler
}

// A HotSwap wraps a HandlerCloser and adds the support for a new handler to
// swapped in the middle of execution with no interruption to processing. In
// this manner, an external listener can load a new configuration, create a new
// Handler, and call Swap. In a threadsafe manner, the old Handler will be
// replaced and all future calls to ServeHTTP will use the new version of the
// handler.
type HotSwap struct {
	mu sync.RWMutex
	hc HandlerCloser
}

// Swap takes a new Pipeline and atomicly exchanges a new handler for all
// future processing. It ensures that all active calls to ServeHTTP are done
// and then closes the old Pipeline
func (h *HotSwap) Swap(hc HandlerCloser) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.hc != nil {
		h.hc.Close()
	}
	h.hc = hc
}

// ServeHTTP delegates incoming requests to the contained handler in a thread
// safe manner.
func (h *HotSwap) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	h.hc.ServeHTTP(w, r)
}

// Close closes the contained HandlerCloser, first ensuring that all in-progress
// calls to ServeHTTP have completed. It is up to the caller to ensure that no
// calls to ServeHTTP are made after this function has started.
func (h *HotSwap) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.hc != nil {
		h.hc.Close()
	}
}
