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

// Annotations lets you attach an arbitrary collection of extra data to each
// individual report, and to each report batch.  Each annotation has a name and
// an arbitrary type; it's up to you to make sure that your processors don't
// make conflicting assumptions about the type of an annotation with a
// particular name.
type Annotations struct {
	Annotations map[string]interface{}
}

// GetAnnotation returns the annotation with the given name, or nil if there
// isn't one.
func (a *Annotations) GetAnnotation(name string) interface{} {
	return a.Annotations[name]
}

// GetOrAddAnnotation returns the annotation with the given name, if it exists.
// If it doesn't, then we save `defaultValue` as the new value for this
// annotation, and return it.
func (a *Annotations) GetOrAddAnnotation(name string, defaultValue interface{}) interface{} {
	result, present := a.Annotations[name]
	if present {
		return result
	}
	a.SetAnnotation(name, defaultValue)
	return defaultValue
}

// SetAnnotation adds an annotation, overwriting any existing annotation with
// the same name.
func (a *Annotations) SetAnnotation(name string, value interface{}) {
	if a.Annotations == nil {
		a.Annotations = make(map[string]interface{})
	}
	a.Annotations[name] = value
}
