// Copyright 2017-2024 Fortio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Moved JSON logging out so it can be skipped for smallest binaries based on build tags.
// This file is the variant that does serialization manually instead of using json.Marshal.

//go:build no_json

package log // import "fortio.org/log"

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

var fullJSON = false

// restore version "manual json serialization" from
// https://github.com/fortio/log/pull/46/files#diff-ff87b7c4777a35588053a509583d66c9f404ccbea9e1c71d2a5f224d7ad1323e
func arrayToString(s []interface{}) string {
	var buf strings.Builder
	buf.WriteString("[")
	for i, e := range s {
		if i != 0 {
			buf.WriteString(",")
		}
		vv := ValueType[interface{}]{Val: e}
		buf.WriteString(vv.String())
	}
	buf.WriteString("]")
	return buf.String()
}

func mapToString(s map[string]interface{}) string {
	var buf strings.Builder
	buf.WriteString("{")
	keys := make([]string, 0, len(s))
	for k := range s {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for i, k := range keys {
		if i != 0 {
			buf.WriteString(",")
		}
		buf.WriteString(fmt.Sprintf("%q", k))
		buf.WriteString(":")
		vv := ValueType[interface{}]{Val: s[k]}
		buf.WriteString(vv.String())
	}
	buf.WriteString("}")
	return buf.String()
}

const nullString = "null"

func (v ValueType[T]) String() string {
	// if the type is numeric, use Sprint(v.val) otherwise use Sprintf("%q", v.Val) to quote it.
	switch s := any(v.Val).(type) {
	case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return fmt.Sprint(s)
	case string:
		return fmt.Sprintf("%q", s)
	case *string:
		if s == nil {
			return nullString
		}
		return fmt.Sprintf("%q", *s)
	case []interface{}:
		return arrayToString(s)
	case map[string]interface{}:
		return mapToString(s)
	case error:
		return fmt.Sprintf("%q", s.Error()) // nil errors handled in case nil below
	case nil:
		return nullString // nil interface{} like `var err error` (but not nil *string etc.)
	default:
		val := reflect.ValueOf(s)
		k := val.Kind()
		if k == reflect.Ptr || k == reflect.Interface {
			if val.IsNil() {
				return nullString
			}
			vv := ValueType[interface{}]{Val: val.Elem().Interface()}
			return vv.String()
		}
		return fmt.Sprintf("%q", fmt.Sprintf("%+v", v.Val))
	}
}
