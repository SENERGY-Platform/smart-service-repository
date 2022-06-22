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

package selectables

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"log"
	"net/http"
	"net/url"
	"runtime/debug"
)

type Selectables struct {
	config configuration.Config
}

func New(config configuration.Config) *Selectables {
	return &Selectables{config: config}
}

func (this *Selectables) Get(token auth.Token, searchedEntities []string, criteria []model.Criteria) (result []model.Selectable, err error, code int) {
	requestBody := new(bytes.Buffer)
	err = json.NewEncoder(requestBody).Encode(criteria)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	var query url.Values = map[string][]string{}
	if len(searchedEntities) == 0 {
		query.Set("include_devices", "true")
		query.Set("include_groups", "true")
		query.Set("include_imports", "true")
	} else {
		for _, searched := range searchedEntities {
			if searched == model.DeviceFilter {
				query.Set("include_devices", "true")
			}
			if searched == model.GroupFilter {
				query.Set("include_groups", "true")
			}
			if searched == model.ImportFilter {
				query.Set("include_imports", "true")
			}
		}
	}

	endpoint := this.config.DeviceSelectionApi + "/v2/query/selectables?" + query.Encode()
	req, err := http.NewRequest("POST", endpoint, requestBody)
	if err != nil {
		log.Println("ERROR: ", err)
		debug.PrintStack()
		return result, err, http.StatusInternalServerError
	}
	req.Header.Set("Authorization", token.Jwt())
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("ERROR: ", err)
		debug.PrintStack()
		return result, err, http.StatusInternalServerError
	}
	if resp.StatusCode >= 300 {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		return result, errors.New(buf.String()), http.StatusInternalServerError
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		log.Println("ERROR: ", err)
		debug.PrintStack()
		return result, err, http.StatusInternalServerError
	}
	return result, nil, http.StatusOK
}
