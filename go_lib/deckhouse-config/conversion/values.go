/*
Copyright 2022 Flant JSC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package conversion

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// JSONValues is a helper to simplify values manipulation in conversion functions.
// Access and update map[string]interface{} is difficult, so gjson and sjson come to the rescue.
type JSONValues struct {
	m          sync.RWMutex
	jsonValues []byte
}

func NewFromMap(in map[string]interface{}) (*JSONValues, error) {
	data, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	return NewFromBytes(data), nil
}

func NewFromBytes(bytes []byte) *JSONValues {
	return &JSONValues{
		jsonValues: bytes,
	}
}

func (v *JSONValues) Get(path string) gjson.Result {
	v.m.RLock()
	defer v.m.RUnlock()
	return gjson.GetBytes(v.jsonValues, path)
}

func (v *JSONValues) Set(path string, value interface{}) error {
	v.m.Lock()
	defer v.m.Unlock()

	newValues, err := sjson.SetBytes(v.jsonValues, path, value)
	if err != nil {
		return err
	}
	v.jsonValues = newValues
	return nil
}

func (v *JSONValues) SetFromJSON(path string, jsonRawValue string) error {
	v.m.Lock()
	defer v.m.Unlock()

	newValues, err := sjson.SetRawBytes(v.jsonValues, path, []byte(jsonRawValue))
	if err != nil {
		return err
	}
	v.jsonValues = newValues
	return nil
}

// Delete removes field by path.
func (v *JSONValues) Delete(path string) error {
	v.m.Lock()
	defer v.m.Unlock()

	newValues, err := sjson.DeleteBytes(v.jsonValues, path)
	if err != nil {
		return err
	}
	v.jsonValues = newValues
	return nil
}

// AsMap transforms values into map[string]interface{} object.
func (v *JSONValues) AsMap() (map[string]interface{}, error) {
	v.m.RLock()
	defer v.m.RUnlock()

	var m map[string]interface{}

	err := json.Unmarshal(v.jsonValues, &m)
	if err != nil {
		return nil, fmt.Errorf("json values to map: %s\n%s", err, string(v.jsonValues))
	}
	return m, nil
}

// Bytes returns underlying json text.
func (v *JSONValues) Bytes() []byte {
	v.m.RLock()
	defer v.m.RUnlock()
	return v.jsonValues
}
