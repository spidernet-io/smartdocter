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

// Moved json logging out so it can be skipped for smallest binaries based on build tags.
// only difference is with nested struct/array logging or logging of types with json Marchaller interface.

//go:build !no_json

package log // import "fortio.org/log"

import (
	"encoding/json"
	"fmt"
	"strconv"
)

var fullJSON = true

func toJSON(v any) string {
	bytes, err := json.Marshal(v)
	if err != nil {
		return strconv.Quote(fmt.Sprintf("ERR marshaling %v: %v", v, err))
	}
	str := string(bytes)
	// We now handle errors before calling toJSON: if there is a marshaller we use it
	// otherwise we use the string from .Error()
	return str
}

func (v ValueType[T]) String() string {
	// if the type is numeric, use Sprint(v.val) otherwise use Sprintf("%q", v.Val) to quote it.
	switch s := any(v.Val).(type) {
	case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return fmt.Sprint(s)
	case string:
		return fmt.Sprintf("%q", s)
	case error:
		// Sadly structured errors like nettwork error don't have the reason in
		// the exposed struct/JSON - ie one gets
		// {"Op":"read","Net":"tcp","Source":{"IP":"127.0.0.1","Port":60067,"Zone":""},
		// "Addr":{"IP":"127.0.0.1","Port":3000,"Zone":""},"Err":{}}
		// instead of
		// read tcp 127.0.0.1:60067->127.0.0.1:3000: i/o timeout
		// Noticed in https://github.com/fortio/fortio/issues/913
		_, hasMarshaller := s.(json.Marshaler)
		if hasMarshaller {
			return toJSON(v.Val)
		}
		return fmt.Sprintf("%q", s.Error())
	/* It's all handled by json fallback now even though slightly more expensive at runtime, it's a lot simpler */
	default:
		return toJSON(v.Val) // was fmt.Sprintf("%q", fmt.Sprint(v.Val))
	}
}
