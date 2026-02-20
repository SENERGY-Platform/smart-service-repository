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
	"io"
	"net/http"
	"runtime/debug"

	"github.com/SENERGY-Platform/permissions-v2/pkg/client"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/notification"
	"github.com/google/uuid"
)

func (this *Controller) AddModule(token auth.Token, instanceId string, module model.SmartServiceModuleInit) (result model.SmartServiceModule, err error, code int) {
	access, err, code := this.permissions.CheckPermission(token.Token, this.config.SmartServiceInstancePermissionsTopic, instanceId, client.Write)
	if err != nil {
		return result, err, code
	}
	if !access {
		return result, errors.New("missing instance write access"), http.StatusForbidden
	}
	return this.addModule(instanceId, module, uuid.NewString())
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
	return this.addModule(businessKey, module, moduleId)
}

func (this *Controller) addModule(instanceId string, module model.SmartServiceModuleInit, moduleId string) (result model.SmartServiceModule, err error, code int) {
	if instanceId == "" {
		return result, errors.New("missing instance id"), http.StatusBadRequest
	}
	userId, err, code := this.getInstanceUserId(instanceId)
	if err != nil {
		return result, err, code
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
	userId := token.GetUserId()
	if token.IsAdmin() {
		userId = ""
	}
	if !token.IsAdmin() && query.InstanceIdFilter != nil {
		access, err, code := this.permissions.CheckPermission(token.Token, this.config.SmartServiceInstancePermissionsTopic, *query.InstanceIdFilter, client.Read)
		if err != nil {
			return nil, err, code
		}
		if !access {
			return nil, errors.New("missing instance read access"), http.StatusForbidden
		}
		userId = "" //instance read access already checked
	}
	if !token.IsAdmin() && query.InstanceIds != nil {
		permResult, err, code := this.permissions.CheckMultiplePermissions(token.Token, this.config.SmartServiceInstancePermissionsTopic, query.InstanceIds, client.Read)
		if err != nil {
			return nil, err, code
		}
		for _, access := range permResult {
			if !access {
				return nil, errors.New("missing instance read access"), http.StatusForbidden
			}
		}
		userId = "" //instance read access already checked
	}
	return this.db.ListModules(userId, query)
}

func (this *Controller) ListModulesOfProcessInstance(processInstanceId string, query model.ModuleQueryOptions) (result []model.SmartServiceModule, err error, code int) {
	businessKey, err, code := this.camunda.GetProcessInstanceBusinessKey(processInstanceId)
	if err != nil {
		return result, err, code
	}
	instance, err, code := this.db.GetInstance(businessKey, "")
	if err != nil {
		return result, err, code
	}
	userId := instance.UserId
	query.InstanceIdFilter = &instance.Id
	return this.db.ListModules(userId, query)
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
		this.config.GetLogger().Error("error in prepareModule", "error", err, "stack", string(debug.Stack()))
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
		err = fmt.Errorf("unexpected response for %v: %v, %v", info.Url, resp.StatusCode, string(temp))
		this.config.GetLogger().Error("error in useModuleDeleteInfo", "error", err, "stack", string(debug.Stack()))
		return err
	}
	_, _ = io.ReadAll(resp.Body)
	return nil
}

func (this *Controller) DeleteModule(token auth.Token, id string, ignoreModuleDeleteError bool) (error, int) {
	module, err, code := this.db.GetModule(id, "")
	if err != nil {
		if code == http.StatusNotFound {
			return nil, http.StatusOK //module is already none-existent
		}
		return err, code
	}
	access, err, code := this.permissions.CheckPermission(token.Token, this.config.SmartServiceInstancePermissionsTopic, module.InstanceId, client.Administrate)
	if err != nil {
		return err, code
	}
	if !access {
		return errors.New("missing instance administrate access"), http.StatusForbidden
	}

	return this.deleteModule(module, ignoreModuleDeleteError)
}

func (this *Controller) deleteModule(module model.SmartServiceModule, ignoreModuleDeleteError bool) (err error, code int) {
	if module.DeleteInfo != nil {
		err = this.useModuleDeleteInfo(*module.DeleteInfo)
		if err != nil && !ignoreModuleDeleteError {
			return err, http.StatusInternalServerError
		}
	}
	return this.db.DeleteModule(module.Id, module.UserId)
}

func (this *Controller) GetModule(token auth.Token, id string) (model.SmartServiceModule, error, int) {
	module, err, code := this.db.GetModule(id, "")
	if err != nil {
		return model.SmartServiceModule{}, err, code
	}
	access, err, code := this.permissions.CheckPermission(token.Token, this.config.SmartServiceInstancePermissionsTopic, module.InstanceId, client.Read)
	if err != nil {
		return model.SmartServiceModule{}, err, code
	}
	if !access {
		return model.SmartServiceModule{}, errors.New("missing instance read access"), http.StatusForbidden
	}
	return this.db.GetModule(id, "")
}

func (this *Controller) SetModuleError(token auth.Token, moduleId string, errMsg string) (error, int) {
	module, err, code := this.db.GetModule(moduleId, "")
	if err != nil {
		return err, code
	}
	instanceId := module.InstanceId
	access, err, code := this.permissions.CheckPermission(token.Token, this.config.SmartServiceInstancePermissionsTopic, instanceId, client.Write)
	if err != nil {
		return err, code
	}
	if !access {
		return errors.New("missing instance write access"), http.StatusForbidden
	}
	if moduleId == "" {
		return errors.New("missing module id"), http.StatusBadRequest
	}

	instance, err, code := this.db.GetInstance(instanceId, "")
	if err != nil {
		return err, code
	}
	if errMsg != "" && instance.Error == "" {
		_ = notification.Send(this.config.NotificationUrl, notification.Message{
			UserId:  instance.UserId,
			Title:   "Smart-Service-Module Error",
			Message: fmt.Sprintf("Smart-Service-Module Error \nInstance-Name: %s \nInstance-ID: %s \nModule-ID: %s \nModule-Type: %s \nError: %s", instance.Name, instanceId, moduleId, module.ModuleType, errMsg),
		}, this.config.GetLogger())
	}

	err = this.db.SetModuleError(moduleId, instance.UserId, errMsg)
	if err != nil {
		return err, http.StatusInternalServerError
	}
	return nil, http.StatusOK
}
