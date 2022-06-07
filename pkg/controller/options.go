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

package controller

import (
	"encoding/json"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"net/http"
)

func (this *Controller) getParamOptions(token auth.Token, desc model.ParameterDescription) (result []model.Option, err error, code int) {
	if desc.Options != nil {
		result = []model.Option{}
		for label, value := range desc.Options {
			result = append(result, model.Option{
				Value: value,
				Label: label,
			})
		}
		return result, nil, http.StatusOK
	}
	if desc.IotDescription != nil {
		return this.getIotOptions(token, desc.IotDescription)
	}
	return nil, nil, http.StatusOK
}

func (this *Controller) getIotOptions(token auth.Token, description *model.IotDescription) ([]model.Option, error, int) {
	selectables, err, code := this.selectables.Get(token, description.TypeFilter, description.Criteria)
	if err != nil {
		return nil, err, code
	}
	return this.selectablesToOptions(selectables)
}

func (this *Controller) selectablesToOptions(selectables []model.Selectable) (result []model.Option, err error, code int) {
	for _, selectable := range selectables {
		if selectable.Device != nil {
			options, err, code := this.selectableToDeviceOptions(selectable)
			if err != nil {
				return nil, err, code
			}
			result = append(result, options...)
		}
		if selectable.DeviceGroup != nil {
			option := model.Option{
				Kind: "Device-Groups",
			}
			option.Label, option.Value, err, code = this.deviceGroupToOptionInfo(*selectable.DeviceGroup)
			if err != nil {
				return nil, err, code
			}
			result = append(result, option)
		}
		if selectable.Import != nil {
			options, err, code := this.selectableToImportOptions(selectable)
			if err != nil {
				return nil, err, code
			}
			result = append(result, options...)
		}
	}
	return result, nil, http.StatusOK
}

func (this *Controller) selectableToDeviceOptions(selectable model.Selectable) (result []model.Option, err error, code int) {
	withServices := false
	serviceCount := len(selectable.Services)
	for _, service := range selectable.Services {
		withServices = true
		withPaths := false
		pathCount := len(selectable.ServicePathOptions[service.Id])
		for _, path := range selectable.ServicePathOptions[service.Id] {
			withPaths = true
			option, err := optionFromDeviceServiceAndPath(*selectable.Device, service, serviceCount, path.Path, pathCount)
			if err != nil {
				return nil, err, http.StatusInternalServerError
			}
			result = append(result, option)
		}
		if !withPaths {
			option, err := optionFromDeviceServiceAndPath(*selectable.Device, service, serviceCount, "", 0)
			if err != nil {
				return nil, err, http.StatusInternalServerError
			}
			result = append(result, option)
		}
	}
	if !withServices {
		option, err := optionFromDeviceServiceAndPath(*selectable.Device, model.Service{}, 0, "", 0)
		if err != nil {
			return nil, err, http.StatusInternalServerError
		}
		result = append(result, option)
	}
	return result, nil, http.StatusOK
}

func (this *Controller) selectableToImportOptions(selectable model.Selectable) (result []model.Option, err error, code int) {
	withPaths := false
	pathCount := len(selectable.ServicePathOptions[selectable.Import.ImportTypeId])
	for _, path := range selectable.ServicePathOptions[selectable.Import.ImportTypeId] {
		withPaths = true
		option, err := optionFromImport(*selectable.Import, path.Path, pathCount)
		if err != nil {
			return nil, err, http.StatusInternalServerError
		}
		result = append(result, option)
	}
	if !withPaths {
		option, err := optionFromImport(*selectable.Import, "", 0)
		if err != nil {
			return nil, err, http.StatusInternalServerError
		}
		result = append(result, option)
	}
	return result, nil, http.StatusOK
}

func optionFromImport(importOption model.Import, path string, lenPaths int) (option model.Option, err error) {
	option.Kind = "Imports"
	option.Label = importOption.Name
	importSelection := model.ImportSelection{
		Id: importOption.Id,
	}
	if path != "" {
		importSelection.Path = &path
		if lenPaths > 1 {
			option.Label = option.Label + " / " + path
		}
	}

	temp, err := json.Marshal(model.IotOption{ImportSelection: &importSelection})
	if err != nil {
		return option, err
	}
	option.Value = string(temp)
	return option, nil
}

func optionFromDeviceServiceAndPath(device model.Device, service model.Service, lenServices int, path string, lenPaths int) (option model.Option, err error) {
	option.Kind = "Devices"
	option.Label = device.DisplayName
	if option.Label == "" {
		option.Label = device.Name
	}
	if option.Label == "" {
		option.Label = device.Id
	}
	deviceSelection := model.DeviceSelection{
		DeviceId: device.Id,
	}
	if service.Id != "" {
		deviceSelection.ServiceId = &service.Id
		if lenServices > 1 {
			option.Label = option.Label + " / " + service.Name
		}
		if path != "" {
			deviceSelection.Path = &path
			if lenPaths > 1 {
				option.Label = option.Label + " / " + path
			}
		}
	}

	temp, err := json.Marshal(model.IotOption{DeviceSelection: &deviceSelection})
	if err != nil {
		return option, err
	}
	option.Value = string(temp)
	return option, nil
}

func (this *Controller) deviceGroupToOptionInfo(group model.DeviceGroup) (label string, encodedValue string, err error, code int) {
	label = group.Name
	temp, err := json.Marshal(model.IotOption{DeviceGroupSelection: &model.DeviceGroupSelection{Id: group.Id}})
	if err != nil {
		return label, encodedValue, err, http.StatusInternalServerError
	}
	return label, string(temp), nil, http.StatusOK
}
