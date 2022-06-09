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

//---------------------------------
// release parameter description, stored in db
//---------------------------------

type ParameterDescription struct {
	Id             string                 `json:"id" bson:"id"`
	Label          string                 `json:"label" bson:"label"`
	Description    string                 `json:"description" bson:"description"`
	Type           string                 `json:"type" bson:"type"`
	DefaultValue   interface{}            `json:"default_value" bson:"default_value"`
	Multiple       bool                   `json:"multiple" bson:"multiple"`
	Options        map[string]interface{} `json:"options,omitempty" bson:"options,omitempty"`
	IotDescription *IotDescription        `json:"iot_description" bson:"iot_description"`
}

type IotDescription struct {
	TypeFilter []FilterPossibility `json:"type_filter" bson:"type_filter"`
	Criteria   []Criteria          `json:"criteria" bson:"criteria"`
}

type Criteria struct {
	Interaction   *Interaction `json:"interaction" bson:"interaction"`
	FunctionId    *string      `json:"function_id" bson:"function_id"`
	DeviceClassId *string      `json:"device_class_id" bson:"device_class_id"`
	AspectId      *string      `json:"aspect_id" bson:"aspect_id"`
}

type FilterPossibility = string

const (
	DeviceFilter FilterPossibility = "device"
	GroupFilter  FilterPossibility = "group"
	ImportFilter FilterPossibility = "import"
)

//---------------------------------
// parameters in api
//---------------------------------

type SmartServiceParameters []SmartServiceParameter

type SmartServiceParameter struct {
	Id    string      `json:"id"`
	Value interface{} `json:"value"`
}

type SmartServiceExtendedParameter struct {
	SmartServiceParameter
	Label        string      `json:"label"`
	Description  string      `json:"description"`
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

type IotOption struct {
	DeviceSelection      *DeviceSelection      `json:"device_selection,omitempty"`
	DeviceGroupSelection *DeviceGroupSelection `json:"device_group_selection,omitempty"`
	ImportSelection      *ImportSelection      `json:"import_selection,omitempty"`
}

type DeviceSelection struct {
	DeviceId  string  `json:"device_id"`
	ServiceId *string `json:"service_id"`
	Path      *string `json:"path"`
}

type DeviceGroupSelection struct {
	Id string `json:"id"`
}

type ImportSelection struct {
	Id   string  `json:"id"`
	Path *string `json:"path"`
}
