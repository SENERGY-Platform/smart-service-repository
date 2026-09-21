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

package selectables

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/SENERGY-Platform/smart-service-repository/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
)

// device-selection is the reader that folds the deprecated aspect id into the list, so what
// this service owes it is both spellings exactly as the design named them. The assertions are
// on the wire format rather than on the model, because that is what the other side decodes.
func TestGetSendsAspectIds(t *testing.T) {
	ptr := func(value string) *string { return &value }

	tests := []struct {
		name     string
		criteria []model.Criteria
		expected []map[string]interface{}
	}{
		{
			name:     "an aspect list is sent as aspect_ids",
			criteria: []model.Criteria{{FunctionId: ptr("fid"), AspectIds: []string{"aid1", "aid2"}}},
			expected: []map[string]interface{}{{
				"interaction":     nil,
				"function_id":     "fid",
				"device_class_id": nil,
				"aspect_id":       nil,
				"aspect_ids":      []interface{}{"aid1", "aid2"},
			}},
		},
		{
			name:     "a deprecated aspect id is sent unfolded, for the reader to alias",
			criteria: []model.Criteria{{FunctionId: ptr("fid"), AspectId: ptr("aid")}},
			expected: []map[string]interface{}{{
				"interaction":     nil,
				"function_id":     "fid",
				"device_class_id": nil,
				"aspect_id":       "aid",
			}},
		},
		{
			name:     "a single element list is sent as a list",
			criteria: []model.Criteria{{AspectIds: []string{"aid"}}},
			expected: []map[string]interface{}{{
				"interaction":     nil,
				"function_id":     nil,
				"device_class_id": nil,
				"aspect_id":       nil,
				"aspect_ids":      []interface{}{"aid"},
			}},
		},
		{
			name: "every criteria of a list keeps its own aspects",
			criteria: []model.Criteria{
				{FunctionId: ptr("f1"), AspectIds: []string{"aid1"}},
				{FunctionId: ptr("f2"), AspectId: ptr("aid2")},
			},
			expected: []map[string]interface{}{
				{"interaction": nil, "function_id": "f1", "device_class_id": nil, "aspect_id": nil, "aspect_ids": []interface{}{"aid1"}},
				{"interaction": nil, "function_id": "f2", "device_class_id": nil, "aspect_id": "aid2"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requests, selectables := deviceSelectionMock(t)
			_, err, code := selectables.Get(context.Background(), auth.Token{}, []string{model.DeviceFilter}, test.criteria)
			if err != nil {
				t.Error(err, code)
				return
			}
			if len(*requests) != 1 {
				t.Error(len(*requests))
				return
			}
			sent := []map[string]interface{}{}
			err = json.Unmarshal((*requests)[0].body, &sent)
			if err != nil {
				t.Error(err)
				return
			}
			if !reflect.DeepEqual(sent, test.expected) {
				t.Error(sent, test.expected)
			}
		})
	}
}

func TestGetUsesTheV2SelectablesEndpoint(t *testing.T) {
	requests, selectables := deviceSelectionMock(t)
	_, err, code := selectables.Get(context.Background(), auth.Token{}, []string{model.DeviceFilter}, []model.Criteria{})
	if err != nil {
		t.Error(err, code)
		return
	}
	if len(*requests) != 1 {
		t.Error(len(*requests))
		return
	}
	if (*requests)[0].path != "/v2/query/selectables" {
		t.Error((*requests)[0].path)
	}
}

type recordedRequest struct {
	path string
	body []byte
}

// deviceSelectionMock stands in for the device-selection service and records what it was asked.
func deviceSelectionMock(t *testing.T) (*[]recordedRequest, *Selectables) {
	t.Helper()
	requests := &[]recordedRequest{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Error(err)
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		*requests = append(*requests, recordedRequest{path: request.URL.Path, body: body})
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`[]`))
	}))
	t.Cleanup(server.Close)
	return requests, New(configuration.Config{DeviceSelectionApi: server.URL})
}
