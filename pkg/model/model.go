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

package model

//cqrs
type SmartServiceRelease struct {
	Id          string `json:"id"`
	DesignId    string `json:"design_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
}

type SmartServiceInstance struct {
	Id               string                  `json:"id"`
	UserId           string                  `json:"user_id"`
	Name             string                  `json:"name"`
	Description      string                  `json:"description"`
	DesignId         string                  `json:"design_id"`
	ReleaseId        string                  `json:"release_id"`
	Ready            bool                    `json:"ready"`
	IncompleteDelete bool                    `json:"incomplete_delete"`
	Parameter        []SmartServiceParameter `json:"parameter"`
}

type SmartServiceParameters []SmartServiceParameter

type SmartServiceParameter struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Value       interface{} `json:"value"`
}

type SmartServiceExtendedParameter struct {
	SmartServiceParameter
	DefaultValue interface{} `json:"default_value"`
	Type         Type        `json:"type"`
	Options      []Option    `json:"options"`  //if null -> "free text/number/etc"
	Multiple     bool        `json:"multiple"` //if true: Value = new([]Type); if false: Value = new(Type);
}

type Option struct {
	Value interface{} `json:"value"`
	Label string      `json:"label"`
	Kind  string      `json:"kind"` //optional helper for ui/app to group options
}

type SmartServiceModuleBase struct {
	Id         string                 `json:"id"`
	UserId     string                 `json:"user_id"`
	InstanceId string                 `json:"instance_id"`
	DesignId   string                 `json:"design_id"`
	ReleaseId  string                 `json:"release_id"`
	ModuleType string                 `json:"module_type"` //"process-deployment" | "analytics" ...
	ModuleData map[string]interface{} `json:"module_data"`
}

type SmartServiceModule struct {
	SmartServiceModuleBase
	DeleteInfo *ModuleDeleteInfo `json:"delete_info"`
}

type ModuleDeleteInfo struct {
	Url    string `json:"url"` //url receives a DELETE request and responds with a status code < 300 if ok; 404 is successful delete
	UserId string `json:"user_id"`
}

type IndexInfo struct {
	Name       string
	FieldNames []string
	IsUnique   bool
}

func (this IndexInfo) GetIndexName() string {
	return this.Name
}

func (this IndexInfo) GetFieldNames() []string {
	return this.FieldNames
}

func (this IndexInfo) Unique() bool {
	return this.IsUnique
}
