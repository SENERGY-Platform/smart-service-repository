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

package mocks

import (
	"github.com/SENERGY-Platform/smart-service-repository/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/controller"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"net/http"
)

type Selectables struct {
	Response []model.Selectable
}

func NewSelectables(response []model.Selectable) *Selectables {
	if response == nil {
		response = []model.Selectable{}
	}
	return &Selectables{Response: response}
}

func (this *Selectables) Get(token auth.Token, searchedEntities []string, criteria []model.Criteria) (result []model.Selectable, err error, code int) {
	result = this.Response
	if len(searchedEntities) == 0 {
		return result, nil, http.StatusOK
	}
	return controller.ListFilter(result, func(s model.Selectable) bool {
		if s.Device != nil && !controller.ListContains(searchedEntities, func(element string) bool { return element == model.DeviceFilter }) {
			return false
		}
		if s.DeviceGroup != nil && !controller.ListContains(searchedEntities, func(element string) bool { return element == model.GroupFilter }) {
			return false
		}
		if s.Import != nil && !controller.ListContains(searchedEntities, func(element string) bool { return element == model.ImportFilter }) {
			return false
		}
		return true
	}), nil, http.StatusOK
}
