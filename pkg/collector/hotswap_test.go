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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/nel-collector/pkg/collector"
)

func TestHotSwapChangesWhichHandlerIsRun(t *testing.T) {
	codeOkHandler := newDummy(http.StatusOK)
	codeBadRequestHandler := newDummy(http.StatusBadRequest)

	var hs collector.HotSwap
	hs.Swap(codeOkHandler)

	response := httptest.NewRecorder()
	hs.ServeHTTP(response, &http.Request{})
	if want := http.StatusOK; response.Code != want {
		t.Errorf("ServeHTTP: got %d, wanted %d", response.Code, want)
	}

	hs.Swap(codeBadRequestHandler)

	response = httptest.NewRecorder()
	hs.ServeHTTP(response, &http.Request{})
	if want := http.StatusBadRequest; response.Code != want {
		t.Errorf("ServeHTTP: got %d, wanted %d", response.Code, want)
	}
}

type dummyHandler struct {
	statusCode int
}

func (d *dummyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(d.statusCode)
}

func newDummy(statusCode int) *dummyHandler {
	var dummy dummyHandler
	dummy.statusCode = statusCode
	return &dummy
}
