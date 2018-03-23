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
	"testing"

	"github.com/google/nel-collector/pkg/collector"
)

func TestAnnotations(t *testing.T) {
	annotations := &collector.Annotations{}

	// An annotation should start off not being present.
	value := annotations.GetAnnotation("test")
	if value != nil {
		t.Errorf("GetAnnotation(%#v) = %#v, wanted nil", "test", value)
	}

	// But we can add it.
	value = annotations.GetOrAddAnnotation("test", "hello world")
	if value != "hello world" {
		t.Errorf("GetOrAddAnnotation(%#v, %#v) = %#v, wanted %#v", "test", "hello world", value, "hello world")
	}

	// And we can get it again.
	value = annotations.GetAnnotation("test")
	if value != "hello world" {
		t.Errorf("GetAnnotation(%#v) = %#v, wanted %#v", "test", value, "hello world")
	}

	// And we can overwrite it.
	annotations.SetAnnotation("test", "goodbye world")
	value = annotations.GetAnnotation("test")
	if value != "goodbye world" {
		t.Errorf("GetAnnotation(%#v) = %#v, wanted %#v", "test", value, "goodbye world")
	}
}
