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

type SmartServiceModel struct {
	Id     string `json:"id"`
	UserId string `json:"user_id"`
}

//cqrs
type SmartServiceDeployment struct {
	Id      string `json:"id"`
	ModelId string `json:"model_id"`
}

type SmartServiceInstance struct {
	Id               string                            `json:"id"`
	UserId           string                            `json:"user_id"`
	ModelId          string                            `json:"model_id"`
	DeploymentId     string                            `json:"deployment_id"`
	Ready            bool                              `json:"ready"`
	IncompleteDelete bool                              `json:"incomplete_delete"`
	Parameter        []SmartServiceDeploymentParameter `json:"parameter"`
	Modules          []SmartServiceModuleBase          `json:"modules"`
}

type SmartServiceDeploymentParameters []SmartServiceDeploymentParameter

type SmartServiceDeploymentParameter struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Value       interface{} `json:"value"`
}

type SmartServiceDeploymentExtendedParameter struct {
	SmartServiceDeploymentParameter
	DefaultValue       interface{}   `json:"default_value"`
	Type               Type          `json:"type"`
	Options            []interface{} `json:"options"`
	IsJsonEncodedArray bool          `json:"is_json_encoded_array"`

	//Validators   []OptionValidator `json:"validators"`
}

type Option struct {
	Value interface{} `json:"value"`
	Label string      `json:"label"`
	Kind  string      `json:"kind"` //optional helper for ui/app to group options
}

/*
type OptionValidator struct {
	Path       string `json:"path"`
	Expression string `json:"expression"`
}
*/

type SmartServiceModuleBase struct {
	Id           string                 `json:"id"`
	UserId       string                 `json:"user_id"`
	InstanceId   string                 `json:"instance_id"`
	ModelId      string                 `json:"model_id"`
	DeploymentId string                 `json:"deployment_id"`
	ModuleType   string                 `json:"module_type"` //"process-deployment" | "analytics" ...
	ModuleData   map[string]interface{} `json:"module_data"`
}

type SmartServiceModule struct {
	SmartServiceModuleBase
	DeleteInfo *ModuleDeleteInfo `json:"delete_info"`
}

type ModuleDeleteInfo struct {
	Url    string `json:"url"` //url receives a DELETE request and responds with a status code < 300 if ok; 404 is successful delete
	UserId string `json:"user_id"`
}
