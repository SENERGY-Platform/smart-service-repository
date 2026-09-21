/*
 * Copyright 2026 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package model

import (
	"encoding/json"
	"reflect"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
)

func ptr[T any](value T) *T {
	return &value
}

// A criteria is written by a design, stored with the release and read by device-selection.
// What matters at each of those boundaries is that both spellings survive unchanged: the fold
// happens on the reading side, so anything dropped here is gone for good.
func TestCriteriaAspectIdsJson(t *testing.T) {
	t.Run("carries the aspect list", func(t *testing.T) {
		encoded := toJsonMap(t, Criteria{FunctionId: ptr("fid"), AspectIds: []string{"aid1", "aid2"}})
		if !reflect.DeepEqual(encoded["aspect_ids"], []interface{}{"aid1", "aid2"}) {
			t.Error(encoded["aspect_ids"])
		}
	})

	t.Run("keeps the aspect list in the order the design named it", func(t *testing.T) {
		encoded := toJsonMap(t, Criteria{AspectIds: []string{"aid2", "aid1"}})
		if !reflect.DeepEqual(encoded["aspect_ids"], []interface{}{"aid2", "aid1"}) {
			t.Error(encoded["aspect_ids"])
		}
	})

	t.Run("omits the aspect list when no aspect is named", func(t *testing.T) {
		encoded := toJsonMap(t, Criteria{FunctionId: ptr("fid")})
		if _, isSet := encoded["aspect_ids"]; isSet {
			t.Error(encoded)
		}
	})

	t.Run("passes the deprecated aspect id on without folding it", func(t *testing.T) {
		encoded := toJsonMap(t, Criteria{AspectId: ptr("aid")})
		if encoded["aspect_id"] != "aid" {
			t.Error(encoded["aspect_id"])
		}
		if _, isSet := encoded["aspect_ids"]; isSet {
			t.Error("a stored criteria must not gain an aspect list it was not written with", encoded)
		}
	})

	t.Run("passes both spellings on side by side", func(t *testing.T) {
		encoded := toJsonMap(t, Criteria{AspectId: ptr("aid1"), AspectIds: []string{"aid2"}})
		if encoded["aspect_id"] != "aid1" {
			t.Error(encoded["aspect_id"])
		}
		if !reflect.DeepEqual(encoded["aspect_ids"], []interface{}{"aid2"}) {
			t.Error(encoded["aspect_ids"])
		}
	})

	t.Run("reads an aspect list a design wrote", func(t *testing.T) {
		criteria := Criteria{}
		err := json.Unmarshal([]byte(`{"function_id":"fid","aspect_ids":["aid1","aid2"]}`), &criteria)
		if err != nil {
			t.Error(err)
			return
		}
		if !reflect.DeepEqual(criteria.AspectIds, []string{"aid1", "aid2"}) {
			t.Error(criteria.AspectIds)
		}
		if criteria.AspectId != nil {
			t.Error(*criteria.AspectId)
		}
	})

	t.Run("reads a deprecated single aspect id a design wrote", func(t *testing.T) {
		criteria := Criteria{}
		err := json.Unmarshal([]byte(`{"function_id":"fid","aspect_id":"aid"}`), &criteria)
		if err != nil {
			t.Error(err)
			return
		}
		if criteria.AspectId == nil || *criteria.AspectId != "aid" {
			t.Error(criteria.AspectId)
		}
		if criteria.AspectIds != nil {
			t.Error(criteria.AspectIds)
		}
	})
}

// The release a criteria sits in is stored in mongo, so the aspect list has to survive bson as
// well. The omitted list is the reason a release written before the lists compares unchanged.
func TestCriteriaAspectIdsBson(t *testing.T) {
	t.Run("survives a round trip", func(t *testing.T) {
		in := Criteria{FunctionId: ptr("fid"), AspectId: ptr("aid1"), AspectIds: []string{"aid2", "aid3"}}
		encoded, err := bson.Marshal(in)
		if err != nil {
			t.Error(err)
			return
		}
		out := Criteria{}
		err = bson.Unmarshal(encoded, &out)
		if err != nil {
			t.Error(err)
			return
		}
		if !reflect.DeepEqual(in, out) {
			t.Error(in, out)
		}
	})

	t.Run("omits the aspect list when no aspect is named", func(t *testing.T) {
		encoded, err := bson.Marshal(Criteria{AspectId: ptr("aid")})
		if err != nil {
			t.Error(err)
			return
		}
		stored := map[string]interface{}{}
		err = bson.Unmarshal(encoded, &stored)
		if err != nil {
			t.Error(err)
			return
		}
		if _, isSet := stored["aspect_ids"]; isSet {
			t.Error(stored)
		}
	})
}

func toJsonMap(t *testing.T, criteria Criteria) map[string]interface{} {
	t.Helper()
	encoded, err := json.Marshal(criteria)
	if err != nil {
		t.Fatal(err)
	}
	result := map[string]interface{}{}
	err = json.Unmarshal(encoded, &result)
	if err != nil {
		t.Fatal(err)
	}
	return result
}
