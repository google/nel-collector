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

import "net/http"

// CORS intercepts CORS OPTIONS requests and sends a fixed set of headers
// that allows all requests
type CORS struct {
	handler http.Handler
}

// ServeHTTP delegates non-OPTIONS requests to the contained handler and
// gives a fixed response for OPTIONS requests that allows all incoming
// CORS requests
func (c *CORS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Method", "POST")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Origin", "*")
	} else {
		c.handler.ServeHTTP(w, r)
	}
}
