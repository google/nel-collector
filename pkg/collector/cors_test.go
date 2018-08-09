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
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeHTTPHandler struct{}

func (f fakeHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "fake response")
}

func TestForwardNonOptionsRequest(t *testing.T) {
	c := CORS{handler: fakeHTTPHandler{}}
	request := httptest.NewRequest("GET", "https://example.com/upload", bytes.NewReader([]byte("")))
	response := httptest.NewRecorder()
	c.ServeHTTP(response, request)
	body, err := ioutil.ReadAll(response.Result().Body)
	if err != nil {
		t.Errorf("ReadAll(reponse.Result().Body): err=%v", err)
	}
	if want := "fake response"; string(body) != want {
		t.Errorf("ServeHTTP(method=GET): got %v, wanted %v", string(body), want)
	}
}
func TestRespondsToOptionsRequest(t *testing.T) {
	c := CORS{fakeHTTPHandler{}}
	request := httptest.NewRequest("OPTIONS", "https://example.com/upload", bytes.NewReader([]byte("")))
	response := httptest.NewRecorder()
	c.ServeHTTP(response, request)
	body, err := ioutil.ReadAll(response.Result().Body)
	if want := ""; string(body) != want && err != nil {
		t.Errorf("ReadAll(response.Result().Body): got %v, wanted \"\" with err %v", string(body), err)
	}
	if want, got := "POST", response.Header().Get("Access-Control-Allow-Method"); got != want {
		t.Errorf("response.Header().Get(\"Access-Control-Allow-Method\"): got %v, want %v", got, want)
	}
	if want, got := "Content-Type", response.Header().Get("Access-Control-Allow-Headers"); got != want {
		t.Errorf("response.Header().Get(\"Access-Control-Allow-Headers\"): got %v, want %v", got, want)
	}
	if want, got := "*", response.Header().Get("Access-Control-Allow-Origin"); got != want {
		t.Errorf("response.Header().Get(\"Access-Control-Allow-Origin\"): got %v, want %v", got, want)
	}
}
