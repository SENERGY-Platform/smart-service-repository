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

type SmartServiceDesign struct {
	Id          string `json:"id" bson:"id"`
	UserId      string `json:"user_id" bson:"user_id"`
	Name        string `json:"name" bson:"name"`
	Description string `json:"description"`
	BpmnXml     string `json:"bpmn_xml" bson:"bpmn_xml"`
	SvgXml      string `json:"svg_xml" bson:"svg_xml"`
	UpdatedAt   int64  `json:"updated_at" bson:"updated_at"` //unix timestamp, set by service on creation
}

// cqrs
type SmartServiceRelease struct {
	Id           string `json:"id" bson:"id"`
	DesignId     string `json:"design_id" bson:"design_id"`
	Name         string `json:"name" bson:"name"`
	Description  string `json:"description" bson:"description"`
	CreatedAt    int64  `json:"created_at" bson:"created_at"` //unix timestamp, set by service on creation
	NewReleaseId string `json:"new_release_id,omitempty"`
	Creator      string `json:"creator"`
}

type SmartServiceReleaseWithUsableFlag struct {
	SmartServiceRelease
	Usable bool `json:"usable"`
}

type SmartServiceReleaseExtended struct {
	SmartServiceRelease `bson:",inline"`
	BpmnXml             string                  `json:"bpmn_xml" bson:"bpmn_xml"`
	SvgXml              string                  `json:"svg_xml" bson:"svg_xml"`
	ParsedInfo          SmartServiceReleaseInfo `json:"parsed_info" bson:"parsed_info"`
	PermissionsInfo     PermissionsInfo         `json:"permissions_info,omitempty" bson:"-"` //optional, set if query parameter permissions_info=true
}

type SmartServiceReleaseExtendedWithUsableFlag struct {
	SmartServiceReleaseExtended
	Usable bool `json:"usable"`
}

type PermissionsInfo struct {
	Shared      bool            `json:"shared"`
	Permissions map[string]bool `json:"permissions"`
}

type SmartServiceReleaseInfo struct {
	ParameterDescriptions []ParameterDescription `json:"parameter_descriptions" bson:"parameter_descriptions"`
	MaintenanceProcedures []MaintenanceProcedure `json:"maintenance_procedures" bson:"maintenance_procedures"`
}

type SmartServiceInstance struct {
	SmartServiceInstanceInit `bson:",inline"`
	Id                       string   `json:"id" bson:"id"`
	UserId                   string   `json:"user_id" bson:"user_id"`
	DesignId                 string   `json:"design_id" bson:"design_id"`
	ReleaseId                string   `json:"release_id" bson:"release_id"`
	NewReleaseId             string   `json:"new_release_id,omitempty"`
	RunningMaintenanceIds    []string `json:"running_maintenance_ids,omitempty"`
	Ready                    bool     `json:"ready" bson:"ready"`
	Deleting                 bool     `json:"deleting,omitempty" bson:"deleting"`
	Error                    string   `json:"error,omitempty" bson:"error"` //is set if module-worker notifies the repository about a error
	CreatedAt                int64    `json:"created_at" bson:"created_at"` //unix timestamp, set by service on creation
	UpdatedAt                int64    `json:"updated_at" bson:"updated_at"` //unix timestamp, set by service on creation
}

type SmartServiceInstanceInit struct {
	SmartServiceInstanceInfo `bson:",inline"`
	Parameters               []SmartServiceParameter `json:"parameters" bson:"parameters"`
}

type SmartServiceInstanceInfo struct {
	Name        string `json:"name" bson:"name"`
	Description string `json:"description" bson:"description"`
}

type SmartServiceInstanceVariable struct {
	InstanceId string      `json:"instance_id" bson:"instance_id"`
	UserId     string      `json:"user_id" bson:"user_id"`
	Name       string      `json:"name" bson:"name"`
	Value      interface{} `json:"value" bson:"value"`
}

type SmartServiceModuleBase struct {
	Id         string `json:"id"`
	UserId     string `json:"user_id" bson:"user_id"`
	InstanceId string `json:"instance_id" bson:"instance_id"`
	DesignId   string `json:"design_id" bson:"design_id"`
	ReleaseId  string `json:"release_id" bson:"release_id"`
}

type SmartServiceModule struct {
	SmartServiceModuleBase `bson:",inline"`
	SmartServiceModuleInit `bson:",inline"`
}

type SmartServiceModuleInitList = []SmartServiceModuleInit

type SmartServiceModuleInit struct {
	DeleteInfo *ModuleDeleteInfo      `json:"delete_info" bson:"delete_info"`
	ModuleType string                 `json:"module_type" bson:"module_type"` //"process-deployment" | "analytics" ...
	ModuleData map[string]interface{} `json:"module_data" bson:"module_data"`
	Keys       []string               `json:"keys" bson:"keys"`
}

type ModuleDeleteInfo struct {
	Url    string `json:"url" bson:"url"` //url receives a DELETE request and responds with a status code < 300 || code == 404 if ok
	UserId string `json:"user_id" bson:"user_id"`
}
