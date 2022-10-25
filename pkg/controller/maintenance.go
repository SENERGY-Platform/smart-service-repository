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
	"net/http"
)

func (this *Controller) GetMaintenanceProceduresOfInstance(token auth.Token, instanceId string) (maintenanceProcedure []model.MaintenanceProcedure, instance model.SmartServiceInstance, release model.SmartServiceReleaseExtended, err error, code int) {
	instance, err, code = this.GetInstance(token, instanceId)
	if err != nil {
		return maintenanceProcedure, instance, release, err, code
	}
	release, err, code = this.GetExtendedRelease(token, instance.ReleaseId)
	if err != nil {
		return maintenanceProcedure, instance, release, err, code
	}
	return release.ParsedInfo.MaintenanceProcedures, instance, release, nil, http.StatusOK
}

func (this *Controller) GetMaintenanceProcedureOfInstance(token auth.Token, instanceId string, publicEventId string) (maintenanceProcedure model.MaintenanceProcedure, instance model.SmartServiceInstance, release model.SmartServiceReleaseExtended, err error, code int) {
	var procedures []model.MaintenanceProcedure
	procedures, instance, release, err, code = this.GetMaintenanceProceduresOfInstance(token, instanceId)
	if err != nil {
		return model.MaintenanceProcedure{}, instance, release, err, code
	}
	for _, procedure := range procedures {
		if procedure.PublicEventId == publicEventId {
			return procedure, instance, release, nil, http.StatusOK
		}
	}
	return model.MaintenanceProcedure{}, instance, release, errors.New("not found"), http.StatusNotFound
}

func (this *Controller) GetMaintenanceProcedureParametersOfInstance(token auth.Token, instanceId string, publicEventId string) ([]model.SmartServiceExtendedParameter, error, int) {
	procedure, _, _, err, code := this.GetMaintenanceProcedureOfInstance(token, instanceId, publicEventId)
	if err != nil {
		return nil, err, code
	}
	return this.parameterDescriptionsToSmartServiceExtendedParameter(token, procedure.ParameterDescriptions)
}

func (this *Controller) StartMaintenanceProcedure(token auth.Token, instanceId string, publicEventId string, parameters model.SmartServiceParameters) (error, int) {
	procedure, instance, release, err, code := this.GetMaintenanceProcedureOfInstance(token, instanceId, publicEventId)
	if err != nil {
		return err, code
	}

	if !instance.Ready {
		return fmt.Errorf("instance init is not ready"), http.StatusBadRequest
	}

	if instance.Error != "" {
		return fmt.Errorf("instance has error state: '%v'", instance.Error), http.StatusBadRequest
	}

	parameterWithInstanceInputs := instance.Parameters
	parameterWithInstanceInputs = append(parameterWithInstanceInputs, parameters...)

	extendedProcedureParameters := append(release.ParsedInfo.ParameterDescriptions, procedure.ParameterDescriptions...)

	paramListWithAutoSelect, err, code := this.appendAutoSelectParams(token, parameterWithInstanceInputs, extendedProcedureParameters)
	if err != nil {
		return err, code
	}

	maintenanceId := uuid.NewString()

	this.cleanupMux.Lock()
	defer this.cleanupMux.Unlock()

	err = this.db.AddToRunningMaintenanceIds(instanceId, maintenanceId)
	if err != nil {
		return err, http.StatusInternalServerError
	}

	err = this.camunda.StartMaintenance(instance.ReleaseId, procedure, maintenanceId, paramListWithAutoSelect)
	if err != nil {
		return err, http.StatusInternalServerError
	}
	return nil, http.StatusOK
}
