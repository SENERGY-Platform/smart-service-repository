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

type Selectable struct {
	Device             *Device                 `json:"device,omitempty"`
	Services           []Service               `json:"services,omitempty"`
	DeviceGroup        *DeviceGroup            `json:"device_group,omitempty"`
	Import             *Import                 `json:"import,omitempty"`
	ServicePathOptions map[string][]PathOption `json:"servicePathOptions,omitempty"`
}

type DeviceGroup struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type Device struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
}

type Service struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type Interaction string

const (
	EVENT             Interaction = "event"
	REQUEST           Interaction = "request"
	EVENT_AND_REQUEST Interaction = "event+request"
)

type Import struct {
	Id           string `json:"id"`
	Name         string `json:"name"`
	ImportTypeId string `json:"import_type_id"`
}

type ImportType struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type PathOption struct {
	Path string `json:"path"`
}
