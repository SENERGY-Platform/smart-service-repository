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
	"errors"
	"github.com/SENERGY-Platform/permissions-v2/pkg/client"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"net/http"
)

func (this *Controller) SetVariable(token auth.Token, variable model.SmartServiceInstanceVariable) (result model.SmartServiceInstanceVariable, err error, code int) {
	access, err, code := this.permissions.CheckPermission(token.Token, this.config.SmartServiceInstancePermissionsTopic, variable.InstanceId, client.Write)
	if err != nil {
		return result, err, code
	}
	if !access {
		return result, errors.New("missing instance write access"), http.StatusForbidden
	}

	userId, err, code := this.getInstanceUserId(variable.InstanceId)
	if err != nil {
		return result, err, code
	}

	if variable.UserId == "" {
		variable.UserId = userId
	}
	if variable.UserId != userId {
		return variable, errors.New("userid of variable and instance do not match"), http.StatusForbidden
	}
	err, code = this.ValidateVariable(variable)
	if err != nil {
		return result, err, code
	}
	return this.db.SetVariable(variable)
}

func (this *Controller) SetVariableForProcessInstance(processInstanceId string, element model.SmartServiceInstanceVariable) (result model.SmartServiceInstanceVariable, err error, code int) {
	if processInstanceId == "" {
		return result, errors.New("missing process instance id"), http.StatusBadRequest
	}
	businessKey, err, code := this.camunda.GetProcessInstanceBusinessKey(processInstanceId)
	if err != nil {
		return result, err, code
	}
	instance, err, code := this.db.GetInstance(businessKey, "")
	if err != nil {
		return result, err, code
	}
	element.InstanceId = instance.Id
	element.UserId = instance.UserId
	return this.db.SetVariable(element)
}

func (this *Controller) SetVariablesMapOfProcessInstance(processInstanceId string, mappedVariableValues map[string]interface{}) (err error, code int) {
	if processInstanceId == "" {
		return errors.New("missing process instance id"), http.StatusBadRequest
	}
	businessKey, err, code := this.camunda.GetProcessInstanceBusinessKey(processInstanceId)
	if err != nil {
		return err, code
	}
	instance, err, code := this.db.GetInstance(businessKey, "")
	if err != nil {
		return err, code
	}
	for key, value := range mappedVariableValues {
		_, err, code = this.db.SetVariable(model.SmartServiceInstanceVariable{
			InstanceId: instance.Id,
			UserId:     instance.UserId,
			Name:       key,
			Value:      value,
		})
		if err != nil {
			return err, code
		}
	}
	return
}

func (this *Controller) GetVariablesMap(token auth.Token, instanceId string, query model.VariableQueryOptions) (map[string]interface{}, error, int) {
	variables, err, code := this.ListVariables(token, instanceId, query)
	if err != nil {
		return nil, err, code
	}
	result := map[string]interface{}{}
	for _, element := range variables {
		result[element.Name] = element.Value
	}
	return result, nil, http.StatusOK
}

func (this *Controller) ListVariables(token auth.Token, instanceId string, query model.VariableQueryOptions) ([]model.SmartServiceInstanceVariable, error, int) {
	access, err, code := this.permissions.CheckPermission(token.Token, this.config.SmartServiceInstancePermissionsTopic, instanceId, client.Read)
	if err != nil {
		return nil, err, code
	}
	if !access {
		return nil, errors.New("missing instance read access"), http.StatusForbidden
	}
	return this.db.ListVariables(instanceId, "", query)
}

func (this *Controller) GetVariablesMapOfProcessInstance(processInstanceId string) (map[string]interface{}, error, int) {
	variables, err, code := this.ListVariablesOfProcessInstance(processInstanceId, model.VariableQueryOptions{
		Limit:  0,
		Offset: 0,
		Sort:   "name.asc",
	})
	if err != nil {
		return nil, err, code
	}
	result := map[string]interface{}{}
	for _, element := range variables {
		result[element.Name] = element.Value
	}
	return result, nil, http.StatusOK
}

func (this *Controller) ListVariablesOfProcessInstance(processInstanceId string, query model.VariableQueryOptions) (result []model.SmartServiceInstanceVariable, err error, code int) {
	businessKey, err, code := this.camunda.GetProcessInstanceBusinessKey(processInstanceId)
	if err != nil {
		return result, err, code
	}
	instance, err, code := this.db.GetInstance(businessKey, "")
	if err != nil {
		return result, err, code
	}
	return this.db.ListVariables(instance.Id, instance.UserId, query)
}

func (this *Controller) ValidateVariable(element model.SmartServiceInstanceVariable) (error, int) {
	if element.Name == "" {
		return errors.New("missing variable name"), http.StatusBadRequest
	}
	if element.UserId == "" {
		return errors.New("missing user id"), http.StatusBadRequest
	}
	if element.InstanceId == "" {
		return errors.New("missing instance id"), http.StatusBadRequest
	}
	return nil, http.StatusOK
}

func (this *Controller) DeleteVariable(token auth.Token, instanceId string, name string) (error, int) {
	access, err, code := this.permissions.CheckPermission(token.Token, this.config.SmartServiceInstancePermissionsTopic, instanceId, client.Administrate)
	if err != nil {
		return err, code
	}
	if !access {
		return errors.New("missing instance administrate access"), http.StatusForbidden
	}
	return this.db.DeleteVariable(instanceId, "", name)
}

func (this *Controller) GetVariable(token auth.Token, instanceId string, name string) (model.SmartServiceInstanceVariable, error, int) {
	access, err, code := this.permissions.CheckPermission(token.Token, this.config.SmartServiceInstancePermissionsTopic, instanceId, client.Read)
	if err != nil {
		return model.SmartServiceInstanceVariable{}, err, code
	}
	if !access {
		return model.SmartServiceInstanceVariable{}, errors.New("missing instance read access"), http.StatusForbidden
	}
	return this.db.GetVariable(instanceId, "", name)
}
