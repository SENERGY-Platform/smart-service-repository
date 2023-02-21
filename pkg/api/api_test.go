/*
 * Copyright (c) 2022 InfAI (CC SES)
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

package api

import (
	"encoding/json"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/configuration"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReflect(t *testing.T) {
	GetRouter(configuration.Config{}, nil)
}

func TestWagger(t *testing.T) {
	s := httptest.NewServer(GetRouter(configuration.Config{}, nil))
	defer s.Close()
	resp, err := http.Get(s.URL + "/doc")
	if err != nil {
		t.Error(err)
		return
	}
	result := map[string]interface{}{}
	temp, _ := io.ReadAll(resp.Body)
	t.Log(string(temp))
	err = json.Unmarshal(temp, &result)
	if err != nil {
		t.Error(err)
		return
	}
	t.Logf("%#v", result)
}
