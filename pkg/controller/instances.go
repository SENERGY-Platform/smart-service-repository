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
	"github.com/SENERGY-Platform/smart-service-repository/pkg/notification"
	"github.com/google/uuid"
	"log"
	"net/http"
	"runtime/debug"
	"sync"
	"time"
)

func (this *Controller) CreateInstance(token auth.Token, releaseId string, instanceInfo model.SmartServiceInstanceInit) (result model.SmartServiceInstance, err error, code int) {
	if instanceInfo.Name == "" {
		return result, errors.New("missing name"), http.StatusBadRequest
	}
	if releaseId == "" {
		return result, errors.New("invalid release id"), http.StatusBadRequest
	}
	access, err := this.permissions.CheckAccess(token, this.config.KafkaSmartServiceReleaseTopic, releaseId, "x")
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	if !access {
		return result, errors.New("missing release access"), http.StatusForbidden
	}
	release, err, code := this.db.GetRelease(releaseId)
	if err != nil {
		return result, err, code
	}
	result = model.SmartServiceInstance{
		SmartServiceInstanceInit: instanceInfo,
		Id:                       uuid.NewString(),
		UserId:                   token.GetUserId(),
		DesignId:                 release.DesignId,
		ReleaseId:                release.Id,
		Ready:                    false,
		Error:                    "",
		UpdatedAt:                time.Now().Unix(),
		CreatedAt:                time.Now().Unix(),
	}
	result.UpdatedAt = time.Now().Unix()
	err, code = this.db.SetInstance(result)
	if err != nil {
		return result, err, code
	}
	err = this.camunda.Start(result)
	if err != nil {
		err2, _ := this.db.DeleteInstance(result.Id, result.UserId)
		if err2 != nil {
			log.Println("ERROR:", err2)
			debug.PrintStack()
		}
		return result, err, http.StatusInternalServerError
	}

	return result, nil, http.StatusOK
}

func (this *Controller) UpdateInstanceInfo(token auth.Token, id string, element model.SmartServiceInstanceInfo) (result model.SmartServiceInstance, err error, code int) {
	if element.Name == "" {
		return result, errors.New("missing name"), http.StatusBadRequest
	}
	result, err, code = this.db.GetInstance(id, token.GetUserId())
	if err != nil {
		return result, err, code
	}
	result.SmartServiceInstanceInfo = element
	result.UpdatedAt = time.Now().Unix()
	err, code = this.db.SetInstance(result)
	return result, err, code
}

func (this *Controller) RedeployInstance(token auth.Token, id string, parameters []model.SmartServiceParameter) (result model.SmartServiceInstance, err error, code int) {
	result, err, code = this.db.GetInstance(id, token.GetUserId())
	if err != nil {
		return result, err, code
	}
	access, err := this.permissions.CheckAccess(token, this.config.KafkaSmartServiceReleaseTopic, result.ReleaseId, "x")
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	if !access {
		return result, errors.New("missing release access"), http.StatusForbidden
	}
	err, code = this.DeleteInstance(token, id, false)
	if err != nil {
		return result, err, code
	}
	result.Ready = false
	result.Error = ""
	result.Parameters = parameters
	result.UpdatedAt = time.Now().Unix()
	err, code = this.db.SetInstance(result)
	if err != nil {
		return result, err, code
	}
	err = this.camunda.Start(result)
	if err != nil {
		result.Error = err.Error()
		return result, err, http.StatusInternalServerError
	}

	return result, nil, http.StatusOK
}

func (this *Controller) ListInstances(token auth.Token, query model.InstanceQueryOptions) (result []model.SmartServiceInstance, err error, code int) {
	result, err, code = this.db.ListInstances(token.GetUserId(), query)
	if err != nil {
		return
	}
	result = this.handleReadyAndErrorFields(result)
	return
}

func (this *Controller) GetInstance(token auth.Token, id string) (result model.SmartServiceInstance, err error, code int) {
	result, err, code = this.db.GetInstance(id, token.GetUserId())
	if err != nil {
		return result, err, code
	}
	result = this.handleReadyAndErrorField(result)
	return result, err, code
}

func (this *Controller) DeleteInstance(token auth.Token, id string, ignoreModuleDeleteError bool) (error, int) {
	_, err, code := this.db.GetInstance(id, token.GetUserId())
	if err != nil {
		if code == http.StatusNotFound {
			return nil, http.StatusOK //instance is already none-existent
		}
		return err, code
	}
	err = this.camunda.StopInstance(id)
	if err != nil {
		return err, http.StatusInternalServerError
	}
	err, code = this.handleModuleDeleteReferencesOfInstance(token, id, ignoreModuleDeleteError)
	if err != nil {
		this.SetInstanceError(token, id, err.Error())
		return err, code
	}
	return this.db.DeleteInstance(id, token.GetUserId())
}

func (this *Controller) handleReadyAndErrorFields(list []model.SmartServiceInstance) []model.SmartServiceInstance {
	for i, e := range list {
		list[i] = this.handleReadyAndErrorField(e)
	}
	return list
}

func (this *Controller) handleReadyAndErrorField(instance model.SmartServiceInstance) model.SmartServiceInstance {
	if instance.Ready {
		return instance
	}
	finished, missing := this.camunda.CheckInstanceReady(instance.Id)
	if missing {
		instance.Error = "missing camunda process instance"
		err, _ := this.db.SetInstance(instance)
		if err != nil {
			log.Println("ERROR:", err)
			debug.PrintStack()
			return instance
		}
	}
	if finished {
		instance.Ready = true
		err, _ := this.db.SetInstance(instance)
		if err != nil {
			log.Println("ERROR:", err)
			debug.PrintStack()
			return instance
		}
	}
	return instance
}

func (this *Controller) SetInstanceError(token auth.Token, instanceId string, errMsg string) (error, int) {
	return this.setInstanceError(token.GetUserId(), instanceId, errMsg)
}

func (this *Controller) setInstanceError(userId string, instanceId string, errMsg string) (error, int) {
	if instanceId == "" {
		return errors.New("missing instance id"), http.StatusBadRequest
	}
	_ = notification.Send(this.config.NotificationUrl, notification.Message{
		UserId:  userId,
		Title:   "Smart-Service-Instance Error (Instance-ID:" + instanceId + ")",
		Message: errMsg,
	})
	err := this.db.SetInstanceError(instanceId, userId, errMsg)
	if err != nil {
		return err, http.StatusInternalServerError
	}
	return nil, http.StatusOK
}

func (this *Controller) SetInstanceErrorByProcessInstanceId(processInstanceId string, errMsg string) (error, int) {
	if processInstanceId == "" {
		return errors.New("missing process instance id"), http.StatusBadRequest
	}
	businessKey, err, code := this.camunda.GetProcessInstanceBusinessKey(processInstanceId)
	if err != nil {
		return err, code
	}
	userId, err, code := this.getInstanceUserId(businessKey)
	if err != nil {
		return err, code
	}
	return this.setInstanceError(userId, businessKey, errMsg)
}

func (this *Controller) GetInstanceUserIdByProcessInstanceId(processInstanceId string) (string, error, int) {
	if processInstanceId == "" {
		return "", errors.New("missing process instance id"), http.StatusBadRequest
	}
	businessKey, err, code := this.camunda.GetProcessInstanceBusinessKey(processInstanceId)
	if err != nil {
		return "", err, code
	}
	return this.getInstanceUserId(businessKey)
}

func (this *Controller) getInstanceUserId(instanceId string) (userId string, err error, code int) {
	instance, err, code := this.db.GetInstance(instanceId, "")
	return instance.UserId, err, code
}

func (this *Controller) handleModuleDeleteReferencesOfInstance(token auth.Token, instanceId string, ignoreModuleDeleteErrors bool) (error, int) {
	modules, err, code := this.db.ListModules(token.GetUserId(), model.ModuleQueryOptions{
		InstanceIdFilter: &instanceId,
	})
	if err != nil {
		return err, code
	}
	wg := sync.WaitGroup{}
	mux := sync.Mutex{}
	for _, m := range modules {
		if m.DeleteInfo != nil {
			deleteInfo := *m.DeleteInfo
			wg.Add(1)
			go func() {
				defer wg.Done()
				tempErr := this.useModuleDeleteInfo(deleteInfo)
				if tempErr != nil && !ignoreModuleDeleteErrors {
					mux.Lock()
					defer mux.Unlock()
					err = tempErr
				}
			}()
		}
	}
	wg.Wait()
	return nil, http.StatusOK
}
