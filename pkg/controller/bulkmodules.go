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
	"github.com/SENERGY-Platform/smart-service-repository/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"github.com/google/uuid"
	"log"
	"net/http"
	"runtime/debug"
)

func (this *Controller) AddModules(token auth.Token, instanceId string, modules []model.SmartServiceModuleInit) (result []model.SmartServiceModule, err error, code int) {
	return this.addModules(token.GetUserId(), instanceId, modules)
}

func (this *Controller) addModules(userId string, instanceId string, modules []model.SmartServiceModuleInit) (result []model.SmartServiceModule, err error, code int) {
	if instanceId == "" {
		return result, errors.New("missing instance id"), http.StatusBadRequest
	}
	elements, err, code := this.prepareModules(userId, instanceId, modules)
	if err != nil {
		return result, err, code
	}
	for _, element := range elements {
		err, code = this.ValidateModule(userId, element)
		if err != nil {
			return result, err, code
		}
	}
	err, code = this.db.SetModules(elements)
	if err != nil {
		return result, err, code
	}
	return elements, nil, http.StatusOK
}

func (this *Controller) AddModulesForProcessInstance(processInstanceId string, modules []model.SmartServiceModuleInit) (result []model.SmartServiceModule, err error, code int) {
	if processInstanceId == "" {
		return result, errors.New("missing process instance id"), http.StatusBadRequest
	}
	businessKey, err, code := this.camunda.GetProcessInstanceBusinessKey(processInstanceId)
	if err != nil {
		return result, err, code
	}
	userId, err, code := this.getInstanceUserId(businessKey)
	if err != nil {
		return result, err, code
	}
	return this.addModules(userId, businessKey, modules)
}

func (this *Controller) prepareModules(userId string, instanceId string, modules []model.SmartServiceModuleInit) (result []model.SmartServiceModule, err error, code int) {
	instance, err, code := this.db.GetInstance(instanceId, userId)
	if err != nil {
		debug.PrintStack()
		log.Println("ERROR:", userId, instanceId, err)
		return result, err, code
	}
	for _, module := range modules {
		result = append(result, model.SmartServiceModule{
			SmartServiceModuleBase: model.SmartServiceModuleBase{
				Id:         uuid.NewString(),
				UserId:     userId,
				InstanceId: instance.Id,
				DesignId:   instance.DesignId,
				ReleaseId:  instance.ReleaseId,
			},
			SmartServiceModuleInit: module,
		})
	}
	return result, nil, http.StatusOK
}
