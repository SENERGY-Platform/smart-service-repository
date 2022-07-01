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
	"fmt"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"github.com/google/uuid"
	"io"
	"log"
	"net/http"
	"runtime/debug"
)

func (this *Controller) AddModule(token auth.Token, instanceId string, module model.SmartServiceModuleInit) (result model.SmartServiceModule, err error, code int) {
	return this.addModule(token.GetUserId(), instanceId, module, uuid.NewString())
}

func (this *Controller) AddModuleForProcessInstance(processInstanceId string, module model.SmartServiceModuleInit) (result model.SmartServiceModule, err error, code int) {
	return this.SetModuleForProcessInstance(processInstanceId, module, uuid.NewString())
}

func (this *Controller) SetModuleForProcessInstance(processInstanceId string, module model.SmartServiceModuleInit, moduleId string) (result model.SmartServiceModule, err error, code int) {
	if moduleId == "" {
		return result, errors.New("missing moduleId"), http.StatusBadRequest
	}
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
	return this.addModule(userId, businessKey, module, moduleId)
}

func (this *Controller) addModule(userId string, instanceId string, module model.SmartServiceModuleInit, moduleId string) (result model.SmartServiceModule, err error, code int) {
	if instanceId == "" {
		return result, errors.New("missing instance id"), http.StatusBadRequest
	}
	element, err, code := this.prepareModule(userId, instanceId, module, moduleId)
	if err != nil {
		return result, err, code
	}
	err, code = this.ValidateModule(userId, element)
	if err != nil {
		return result, err, code
	}
	err, code = this.db.SetModule(element)
	if err != nil {
		return result, err, code
	}
	return this.db.GetModule(element.Id, userId)
}

func (this *Controller) ListModules(token auth.Token, query model.ModuleQueryOptions) ([]model.SmartServiceModule, error, int) {
	return this.db.ListModules(token.GetUserId(), query)
}

func (this *Controller) ValidateModule(userId string, element model.SmartServiceModule) (error, int) {
	if element.Id == "" {
		return errors.New("missing id"), http.StatusBadRequest
	}
	if element.UserId == "" {
		return errors.New("missing user id"), http.StatusBadRequest
	}
	if element.InstanceId == "" {
		return errors.New("missing instance id"), http.StatusBadRequest
	}
	instance, err, code := this.db.GetInstance(element.InstanceId, userId)
	if err != nil {
		if code == http.StatusNotFound {
			code = http.StatusBadRequest
		}
		return fmt.Errorf("referenced smart service instance (%v, %v) not found: %w", element.InstanceId, userId, err), code
	}
	if instance.UserId != element.UserId {
		return errors.New("referenced smart service instance is owned by a different user"), http.StatusForbidden
	}
	return nil, http.StatusOK
}

func (this *Controller) prepareModule(userId string, instanceId string, module model.SmartServiceModuleInit, moduleId string) (result model.SmartServiceModule, err error, code int) {
	instance, err, code := this.db.GetInstance(instanceId, userId)
	if err != nil {
		debug.PrintStack()
		log.Println("ERROR:", userId, instanceId, err)
		return result, err, code
	}
	if moduleId == "" {
		moduleId = uuid.NewString()
	}
	result = model.SmartServiceModule{
		SmartServiceModuleBase: model.SmartServiceModuleBase{
			Id:         moduleId,
			UserId:     userId,
			InstanceId: instance.Id,
			DesignId:   instance.DesignId,
			ReleaseId:  instance.ReleaseId,
		},
		SmartServiceModuleInit: module,
	}

	return result, nil, http.StatusOK
}

func (this *Controller) useModuleDeleteInfo(info model.ModuleDeleteInfo) error {
	req, err := http.NewRequest("DELETE", info.Url, nil)
	if err != nil {
		return err
	}
	if info.UserId != "" {
		token, err := this.userTokenProvider(info.UserId)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", token.Jwt())
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 && resp.StatusCode != http.StatusNotFound {
		temp, _ := io.ReadAll(resp.Body)
		err = fmt.Errorf("unexpected response: %v, %v", resp.StatusCode, string(temp))
		log.Println("ERROR:", err)
		debug.PrintStack()
		return err
	}
	_, _ = io.ReadAll(resp.Body)
	return nil
}

func (this *Controller) DeleteModule(token auth.Token, id string, ignoreModuleDeleteError bool) (error, int) {
	module, err, code := this.db.GetModule(id, token.GetUserId())
	if err != nil {
		if code == http.StatusNotFound {
			return nil, http.StatusOK //module is already none-existent
		}
		return err, code
	}
	if module.DeleteInfo != nil {
		err = this.useModuleDeleteInfo(*module.DeleteInfo)
		if err != nil && !ignoreModuleDeleteError {
			return err, http.StatusInternalServerError
		}
	}
	return this.db.DeleteModule(id, token.GetUserId())
}
