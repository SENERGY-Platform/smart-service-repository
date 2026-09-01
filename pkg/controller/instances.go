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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/SENERGY-Platform/permissions-v2/pkg/client"
	perm_model "github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/notification"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/tracing"
	"github.com/google/uuid"
)

func (this *Controller) CreateInstance(ctx context.Context, token auth.Token, releaseId string, instanceInfo model.SmartServiceInstanceInit) (result model.SmartServiceInstance, err error, code int) {
	if instanceInfo.Name == "" {
		return result, errors.New("missing name"), http.StatusBadRequest
	}
	if releaseId == "" {
		return result, errors.New("invalid release id"), http.StatusBadRequest
	}
	access, err, _ := this.permissions.CheckPermissionContext(ctx, token.Jwt(), this.config.SmartServiceReleasePermissionsTopic, releaseId, client.Execute)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	if !access {
		return result, errors.New("missing release access"), http.StatusForbidden
	}
	release, err, code := this.db.GetRelease(ctx, releaseId, false)
	if err != nil {
		return result, err, code
	}

	paramListWithoutAutoSelect := instanceInfo.Parameters

	paramListWithAutoSelect, err, code := this.appendAutoSelectParams(ctx, token, instanceInfo.Parameters, release.ParsedInfo.ParameterDescriptions)
	if err != nil {
		return result, err, code
	}

	//store without auto_select_all parameter
	result = model.SmartServiceInstance{
		SmartServiceInstanceInit: instanceInfo,
		Id:                       uuid.NewString(),
		UserId:                   token.GetUserId(),
		DesignId:                 release.DesignId,
		ReleaseId:                release.Id,
		Ready:                    false,
		Error:                    "",
		NewReleaseId:             release.NewReleaseId,
		UpdatedAt:                time.Now().Unix(),
		CreatedAt:                time.Now().Unix(),
	}
	result.UpdatedAt = time.Now().Unix()

	//from here on the instance-id is known: put it into the baggage, so that every following
	//request and log record of this smart-service creation carries it
	var baggageErr error
	ctx, baggageErr = tracing.AddToBaggage(ctx, tracing.BaggageKeyInstanceId, result.Id)
	if baggageErr != nil {
		//telemetry must not prevent the creation of a smart-service
		this.config.GetLogger().WarnContext(ctx, "unable to add instance id to baggage", "error", baggageErr, "instanceId", result.Id)
	}

	this.cleanupMux.Lock()
	defer this.cleanupMux.Unlock()

	_, err, code = this.permissions.SetPermissionContext(ctx, client.InternalAdminToken, this.config.SmartServiceInstancePermissionsTopic, result.Id, client.ResourcePermissions{
		UserPermissions: map[string]client.PermissionsMap{
			result.UserId: {
				Read:         true,
				Write:        true,
				Execute:      true,
				Administrate: true,
			},
		},
	})
	if err != nil {
		return result, err, code
	}

	err, code = this.db.SetInstance(ctx, result)
	if err != nil {
		return result, err, code
	}

	//start with auto_select_all parameter
	result.SmartServiceInstanceInit.Parameters = paramListWithAutoSelect

	err = this.storeInstanceStartVariables(ctx, result)
	if err != nil {
		err2, _ := this.db.DeleteInstance(ctx, result.Id, "")
		if err2 != nil {
			this.config.GetLogger().ErrorContext(ctx, "error in CreateInstance", "error", err2, "stack", string(debug.Stack()))
		}
		return result, err, http.StatusInternalServerError
	}

	result.SmartServiceInstanceInit.Parameters = this.replaceLongParameterWithVariableReference(ctx, paramListWithAutoSelect)

	err = this.camunda.Start(ctx, result)
	if err != nil {
		err2, _ := this.db.DeleteInstance(ctx, result.Id, "")
		if err2 != nil {
			this.config.GetLogger().ErrorContext(ctx, "error in CreateInstance", "error", err2, "stack", string(debug.Stack()))
		}
		return result, err, http.StatusInternalServerError
	}

	//return result without auto_select_all parameters
	result.SmartServiceInstanceInit.Parameters = paramListWithoutAutoSelect
	result.PermissionsInfo = model.PermissionsInfo{
		Shared: false,
		Permissions: map[string]bool{
			"read":         true,
			"write":        true,
			"execute":      true,
			"administrate": true,
		},
	}
	return result, nil, http.StatusOK
}

func (this *Controller) appendAutoSelectParams(ctx context.Context, token auth.Token, parameters []model.SmartServiceParameter, paramDescriptions []model.ParameterDescription) (result []model.SmartServiceParameter, err error, code int) {
	result = []model.SmartServiceParameter{}
	result = append(result, parameters...)
	for _, param := range paramDescriptions {
		if param.AutoSelectAll {
			options, err, code := this.getParamOptions(ctx, token, param)
			if err != nil {
				return result, err, code
			}

			value := []interface{}{}
			for _, option := range options {
				value = append(value, option.Value)
			}

			result = append(result, model.SmartServiceParameter{
				Id:         param.Id,
				Value:      value,
				Label:      param.Label,
				ValueLabel: param.Label,
			})
		}
	}
	return result, nil, http.StatusOK
}

func (this *Controller) UpdateInstanceInfo(ctx context.Context, token auth.Token, id string, element model.SmartServiceInstanceInfo) (result model.SmartServiceInstance, err error, code int) {
	access, err, code := this.permissions.CheckPermissionContext(ctx, token.Token, this.config.SmartServiceInstancePermissionsTopic, id, client.Write)
	if err != nil {
		return result, err, code
	}
	if !access {
		return result, errors.New("missing instance write access"), http.StatusForbidden
	}
	if element.Name == "" {
		return result, errors.New("missing name"), http.StatusBadRequest
	}
	result, err, code = this.db.GetInstance(ctx, id, "")
	if err != nil {
		return result, err, code
	}
	result.SmartServiceInstanceInfo = element
	result.UpdatedAt = time.Now().Unix()
	err, code = this.db.SetInstance(ctx, result)
	if err != nil {
		return result, err, code
	}
	arr := []model.SmartServiceInstance{result}
	err, code = this.fillPermissions(ctx, token, arr)
	if err != nil {
		return result, err, code
	}
	if len(arr) == 1 { // sanity check
		result = arr[0]
	}
	return result, err, code
}

func (this *Controller) RedeployInstance(ctx context.Context, token auth.Token, id string, parameters []model.SmartServiceParameter, releaseId string) (result model.SmartServiceInstance, err error, code int) {
	access, err, code := this.permissions.CheckPermissionContext(ctx, token.Token, this.config.SmartServiceInstancePermissionsTopic, id, client.Administrate)
	if err != nil {
		return result, err, code
	}
	if !access {
		return result, errors.New("missing instance administrate access"), http.StatusForbidden
	}

	result, err, code = this.db.GetInstance(ctx, id, "")
	if err != nil {
		return result, err, code
	}
	access, err, _ = this.permissions.CheckPermissionContext(ctx, token.Jwt(), this.config.SmartServiceReleasePermissionsTopic, result.ReleaseId, client.Execute)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	if !access {
		return result, errors.New("missing release access"), http.StatusForbidden
	}
	err, code = this.DeleteInstance(ctx, token, id, false)
	if err != nil {
		return result, err, code
	}
	result.Ready = false
	result.Deleting = false
	result.Error = ""
	result.Parameters = parameters
	result.UpdatedAt = time.Now().Unix()

	var release model.SmartServiceReleaseExtended
	if releaseId != "" {
		release, err, code = this.GetExtendedRelease(ctx, token, releaseId)
		if err != nil {
			return result, err, code
		}
		result.ReleaseId = release.Id
		if result.NewReleaseId == release.Id {
			result.NewReleaseId = ""
		}
		result.DesignId = release.DesignId
		result.NewReleaseId = release.NewReleaseId
	} else {
		release, err, code = this.GetExtendedRelease(ctx, token, result.ReleaseId)
		if err != nil {
			return result, err, code
		}
	}

	paramListWithoutAutoSelect := result.Parameters

	paramListWithAutoSelect, err, code := this.appendAutoSelectParams(ctx, token, result.Parameters, release.ParsedInfo.ParameterDescriptions)
	if err != nil {
		return result, err, code
	}

	//store without auto_select_all parameter
	result.Parameters = paramListWithoutAutoSelect

	this.cleanupMux.Lock()
	defer this.cleanupMux.Unlock()

	_, err, code = this.permissions.SetPermissionContext(ctx, client.InternalAdminToken, this.config.SmartServiceInstancePermissionsTopic, result.Id, client.ResourcePermissions{
		UserPermissions: map[string]client.PermissionsMap{
			result.UserId: {
				Read:         true,
				Write:        true,
				Execute:      true,
				Administrate: true,
			},
		},
	})
	if err != nil {
		return result, err, code
	}

	err, code = this.db.SetInstance(ctx, result)
	if err != nil {
		return result, err, code
	}

	//start with auto_select_all parameter
	result.SmartServiceInstanceInit.Parameters = paramListWithAutoSelect

	err = this.storeInstanceStartVariables(ctx, result)
	if err != nil {
		err2, _ := this.db.DeleteInstance(ctx, result.Id, result.UserId)
		if err2 != nil {
			this.config.GetLogger().ErrorContext(ctx, "error in CreateInstance", "error", err2, "stack", string(debug.Stack()))
		}
		return result, err, http.StatusInternalServerError
	}

	result.SmartServiceInstanceInit.Parameters = this.replaceLongParameterWithVariableReference(ctx, paramListWithAutoSelect)

	err = this.camunda.Start(ctx, result)
	if err != nil {
		this.config.GetLogger().ErrorContext(ctx, "error in RedeployInstance", "error", err)
		result.Error = err.Error()
		return result, err, http.StatusInternalServerError
	}

	arr := []model.SmartServiceInstance{result}
	err, code = this.fillPermissions(ctx, token, arr)
	if err != nil {
		return result, err, code
	}
	if len(arr) == 1 { // sanity check
		result = arr[0]
	}
	return result, nil, http.StatusOK
}

func (this *Controller) ListInstances(ctx context.Context, token auth.Token, query model.InstanceQueryOptions) (result []model.SmartServiceInstance, total int64, err error, code int) {
	listOptions := client.ListOptions{}
	if len(query.IDs) > 0 {
		listOptions.Ids = query.IDs
	}
	accessibleIds, err, code := this.permissions.ListAccessibleResourceIdsContext(ctx, token.Token, this.config.SmartServiceInstancePermissionsTopic, listOptions, client.Read)
	if err != nil {
		return result, total, err, code
	}
	if len(accessibleIds) == 0 {
		return result, 0, nil, http.StatusOK
	}
	query.IDs = accessibleIds
	result, total, err, code = this.db.ListInstances(ctx, "", query)
	if err != nil {
		return
	}
	result = this.handleReadyAndErrorFields(ctx, result)
	err, code = this.fillPermissions(ctx, token, result)
	if err != nil {
		return result, total, err, code
	}
	return
}

func (this *Controller) GetInstance(ctx context.Context, token auth.Token, id string) (result model.SmartServiceInstance, err error, code int) {
	access, err, code := this.permissions.CheckPermissionContext(ctx, token.Token, this.config.SmartServiceInstancePermissionsTopic, id, client.Read)
	if err != nil {
		return result, err, code
	}
	if !access {
		return result, errors.New("missing instance read access"), http.StatusForbidden
	}
	result, err, code = this.db.GetInstance(ctx, id, "")
	if err != nil {
		return result, err, code
	}
	result = this.handleReadyAndErrorField(ctx, result)
	result = this.removeFinishedMaintenanceIds(ctx, result)
	arr := []model.SmartServiceInstance{result}
	err, code = this.fillPermissions(ctx, token, arr)
	if err != nil {
		return result, err, code
	}
	if len(arr) == 1 { // sanity check
		result = arr[0]
	}
	return result, err, code
}

func (this *Controller) DeleteInstance(ctx context.Context, token auth.Token, id string, ignoreModuleDeleteError bool) (error, int) {
	access, err, code := this.permissions.CheckPermissionContext(ctx, token.Token, this.config.SmartServiceInstancePermissionsTopic, id, client.Administrate, client.Write)
	if err != nil {
		return err, code
	}
	if !access {
		return errors.New("missing instance administrate and/or write access (requires both)"), http.StatusForbidden
	}

	current, err, code := this.db.GetInstance(ctx, id, "")
	if err != nil {
		if code == http.StatusNotFound {
			return nil, http.StatusOK //instance is already none-existent
		}
		return err, code
	}

	//mark instance as transitioning while other delete work is done
	current.Deleting = true
	current.UpdatedAt = time.Now().Unix()
	err, code = this.db.SetInstance(ctx, current)
	if err != nil {
		return err, code
	}

	//stop running instances
	err = this.camunda.StopInstance(ctx, id)
	if err != nil {
		this.SetInstanceError(ctx, token, id, err.Error())
		return err, http.StatusInternalServerError
	}

	//handle module delete infos
	err, code = this.handleModuleDeleteReferencesOfInstance(ctx, id, ignoreModuleDeleteError)
	if err != nil {
		this.SetInstanceError(ctx, token, id, err.Error())
		return err, code
	}

	//delete instance and modules from database
	err, code = this.db.DeleteInstance(ctx, id, "")
	if err != nil {
		this.SetInstanceError(ctx, token, id, err.Error())
		return err, code
	}
	return err, code
}

func (this *Controller) handleReadyAndErrorFields(ctx context.Context, list []model.SmartServiceInstance) []model.SmartServiceInstance {
	for i, e := range list {
		list[i] = this.handleReadyAndErrorField(ctx, e)
	}
	return list
}

const ErrMissingCamundaProcessInstance = "missing camunda process instance"

func (this *Controller) handleReadyAndErrorField(ctx context.Context, instance model.SmartServiceInstance) model.SmartServiceInstance {
	if instance.Ready {
		return instance
	}
	finished, missing, err := this.camunda.CheckInstanceReady(ctx, instance.Id)
	if err != nil {
		this.config.GetLogger().ErrorContext(ctx, "error in handleReadyAndErrorField", "error", err, "stack", string(debug.Stack()))
		return instance
	}
	if missing {
		instance.Ready = false
		instance.Error = ErrMissingCamundaProcessInstance
		err, _ = this.db.SetInstance(ctx, instance)
		if err != nil {
			this.config.GetLogger().ErrorContext(ctx, "error in handleReadyAndErrorField", "error", err, "stack", string(debug.Stack()))
			return instance
		}
	}
	if finished {
		instance.Ready = true
		if instance.Error == ErrMissingCamundaProcessInstance {
			instance.Error = ""
		}
		err, _ := this.db.SetInstance(ctx, instance)
		if err != nil {
			this.config.GetLogger().ErrorContext(ctx, "error in handleReadyAndErrorField", "error", err, "stack", string(debug.Stack()))
			return instance
		}
	}
	return instance
}

func (this *Controller) removeFinishedMaintenanceIds(ctx context.Context, instance model.SmartServiceInstance) model.SmartServiceInstance {
	if len(instance.RunningMaintenanceIds) == 0 {
		return instance
	}
	removedMaintenanceIds := []string{}
	newMaintenanceIds := []string{}
	for _, id := range instance.RunningMaintenanceIds {
		finished, missing, err := this.camunda.CheckInstanceReady(ctx, id)
		if err != nil {
			this.config.GetLogger().ErrorContext(ctx, "error in removeFinishedMaintenanceIds", "error", err, "stack", string(debug.Stack()))
			return instance
		}
		if finished && !missing {
			err = this.camunda.StopInstance(ctx, id)
			if err != nil {
				this.config.GetLogger().ErrorContext(ctx, "error in removeFinishedMaintenanceIds", "error", err, "stack", string(debug.Stack()))
			}
		}
		if missing || finished {
			removedMaintenanceIds = append(removedMaintenanceIds, id)
		} else {
			newMaintenanceIds = append(newMaintenanceIds, id)
		}
	}
	instance.RunningMaintenanceIds = newMaintenanceIds
	if len(removedMaintenanceIds) > 0 {
		err := this.db.RemoveFromRunningMaintenanceIds(ctx, instance.Id, removedMaintenanceIds)
		if err != nil {
			this.config.GetLogger().ErrorContext(ctx, "error in removeFinishedMaintenanceIds", "error", err, "stack", string(debug.Stack()))
			return instance
		}
	}
	return instance
}

func (this *Controller) SetInstanceError(ctx context.Context, token auth.Token, instanceId string, errMsg string) (error, int) {
	access, err, code := this.permissions.CheckPermissionContext(ctx, token.Token, this.config.SmartServiceInstancePermissionsTopic, instanceId, client.Write)
	if err != nil {
		return err, code
	}
	if !access {
		return errors.New("missing instance write access"), http.StatusForbidden
	}
	return this.setInstanceError(ctx, instanceId, errMsg)
}

func (this *Controller) setInstanceError(ctx context.Context, instanceId string, errMsg string) (error, int) {
	if instanceId == "" {
		return errors.New("missing instance id"), http.StatusBadRequest
	}

	instance, err, code := this.db.GetInstance(ctx, instanceId, "")
	if err != nil {
		return err, code
	}

	_ = notification.Send(ctx, this.config.NotificationUrl, notification.Message{
		UserId:  instance.UserId,
		Title:   "Smart-Service-Instance Error",
		Message: fmt.Sprintf("Smart-Service-Instance Error \nInstance-Name: %s \nInstance-ID: %s \nError: %s", instance.Name, instanceId, errMsg),
	}, this.config.GetLogger())
	err = this.db.SetInstanceError(ctx, instanceId, instance.UserId, errMsg)
	if err != nil {
		return err, http.StatusInternalServerError
	}
	return nil, http.StatusOK
}

func (this *Controller) SetInstanceErrorByProcessInstanceId(ctx context.Context, processInstanceId string, errMsg string) (error, int) {
	if processInstanceId == "" {
		return errors.New("missing process instance id"), http.StatusBadRequest
	}
	businessKey, err, code := this.camunda.GetProcessInstanceBusinessKey(ctx, processInstanceId)
	if err != nil {
		return err, code
	}
	return this.setInstanceError(ctx, businessKey, errMsg)
}

func (this *Controller) GetInstanceByProcessInstanceId(ctx context.Context, processInstanceId string) (result model.SmartServiceInstance, err error, code int) {
	if processInstanceId == "" {
		return result, errors.New("missing process instance id"), http.StatusBadRequest
	}
	businessKey, err, code := this.camunda.GetProcessInstanceBusinessKey(ctx, processInstanceId)
	if err != nil {
		return result, err, code
	}
	return this.db.GetInstance(ctx, businessKey, "")
}

func (this *Controller) GetInstanceUserIdByProcessInstanceId(ctx context.Context, processInstanceId string) (string, error, int) {
	if processInstanceId == "" {
		return "", errors.New("missing process instance id"), http.StatusBadRequest
	}
	businessKey, err, code := this.camunda.GetProcessInstanceBusinessKey(ctx, processInstanceId)
	if err != nil {
		return "", err, code
	}
	return this.getInstanceUserId(ctx, businessKey)
}

func (this *Controller) getInstanceUserId(ctx context.Context, instanceId string) (userId string, err error, code int) {
	instance, err, code := this.db.GetInstance(ctx, instanceId, "")
	return instance.UserId, err, code
}

func (this *Controller) handleModuleDeleteReferencesOfInstance(ctx context.Context, instanceId string, ignoreModuleDeleteErrors bool) (error, int) {
	modules, err, code := this.db.ListModules(ctx, "", model.ModuleQueryOptions{
		InstanceIdFilter: &instanceId,
	})
	if err != nil {
		return err, code
	}
	wg := sync.WaitGroup{}
	mux := sync.Mutex{}
	errList := []error{}

	for _, m := range modules {
		if m.DeleteInfo != nil {
			deleteInfo := *m.DeleteInfo
			wg.Add(1)
			go func() {
				defer wg.Done()
				tempErr := this.useModuleDeleteInfo(ctx, deleteInfo)
				if tempErr != nil && !ignoreModuleDeleteErrors {
					mux.Lock()
					defer mux.Unlock()
					errList = append(errList, tempErr)
				}
			}()
		}
	}
	wg.Wait()
	err = errors.Join(errList...)
	if err != nil {
		return err, http.StatusInternalServerError
	}
	return nil, http.StatusOK
}

func (this *Controller) storeInstanceStartVariables(ctx context.Context, result model.SmartServiceInstance) (err error) {
	for _, param := range result.Parameters {
		_, err, _ = this.db.SetVariable(ctx, model.SmartServiceInstanceVariable{
			InstanceId: result.Id,
			UserId:     result.UserId,
			Name:       param.Id,
			Value:      param.Value,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (this *Controller) replaceLongParameterWithVariableReference(ctx context.Context, params []model.SmartServiceParameter) []model.SmartServiceParameter {
	for i, param := range params {
		switch v := param.Value.(type) {
		case string:
			if len(v) >= 3000 {
				param.Value = createRef(param.Id)
				params[i] = param
			}
		default:
			temp, err := json.Marshal(v)
			if err != nil {
				this.config.GetLogger().ErrorContext(ctx, "error in replaceLongParameterWithVariableReference", "error", err, "stack", string(debug.Stack()))
				continue
			}
			if len(string(temp)) >= 3000 {
				param.Value = createRef(param.Id)
				params[i] = param
			}
		}
	}
	return params
}

func (this *Controller) fillPermissions(ctx context.Context, token auth.Token, instances []model.SmartServiceInstance) (err error, code int) {
	ids := []string{}
	for _, instance := range instances {
		ids = append(ids, instance.Id)
	}

	perms, err, code := this.permissions.ListComputedPermissionsContext(ctx, token.Token, this.config.SmartServiceInstancePermissionsTopic, ids)
	if err != nil {
		return err, code
	}

	slices.SortFunc(perms, func(a, b perm_model.ComputedPermissions) int {
		return strings.Compare(a.Id, b.Id)
	})

	for i := range instances {
		j, ok := slices.BinarySearchFunc(perms, instances[i].Id, func(a perm_model.ComputedPermissions, b string) int {
			return strings.Compare(a.Id, b)
		})
		if !ok {
			continue
		}
		perm := perms[j]
		instances[i].PermissionsInfo = model.PermissionsInfo{
			Shared: instances[i].UserId != token.GetUserId(),
			Permissions: map[string]bool{
				"read":         perm.Read,
				"write":        perm.Write,
				"execute":      perm.Execute,
				"administrate": perm.Administrate,
			},
		}
	}
	return nil, http.StatusOK
}

func createRef(id string) string {
	return "{{." + id + "}}"
}
