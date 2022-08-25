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
	"encoding/json"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/permissions"
)

type Permissions struct {
	queryFindResponses map[string]map[string]interface{}
}

func NewPermissions() *Permissions {
	return &Permissions{
		queryFindResponses: map[string]map[string]interface{}{},
	}
}

func (this *Permissions) CheckAccess(token auth.Token, topic string, id string, right string) (bool, error) {
	return true, nil
}

func (this *Permissions) Query(token string, query permissions.QueryMessage, result interface{}) (err error, code int) {
	if _, ok := this.queryFindResponses[token]; !ok {
		this.queryFindResponses[token] = map[string]interface{}{}
	}
	if _, ok := this.queryFindResponses[token][query.Resource]; !ok {
		this.queryFindResponses[token][query.Resource] = []interface{}{}
	}
	temp, _ := json.Marshal(this.queryFindResponses[token][query.Resource])
	err = json.Unmarshal(temp, result)
	return
}

func (this *Permissions) SetQueryFindResponses(queryFindResponses map[string]map[string]interface{}) {
	this.queryFindResponses = queryFindResponses
}

func (this *Permissions) GetResourceRights(token string, topic string, id string) (rights permissions.ResourceRights, err error, code int) {
	return rights, nil, 200
}

func (this *Permissions) SetResourceRights(token string, topic string, id string, rights permissions.ResourceRights, kafkaKey string) error {
	return nil
}
