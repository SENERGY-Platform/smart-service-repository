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
	"net/http"
)

func (this *Controller) GetMaintenanceProceduresOfInstance(token auth.Token, instanceId string) ([]model.MaintenanceProcedure, error, int) {
	instance, err, code := this.GetInstance(token, instanceId)
	if err != nil {
		return nil, err, code
	}
	release, err, code := this.GetExtendedRelease(token, instance.ReleaseId)
	if err != nil {
		return nil, err, code
	}
	return release.ParsedInfo.MaintenanceProcedures, nil, http.StatusOK
}

func (this *Controller) GetMaintenanceProcedureOfInstance(token auth.Token, instanceId string, publicEventId string) (model.MaintenanceProcedure, error, int) {
	procedures, err, code := this.GetMaintenanceProceduresOfInstance(token, instanceId)
	if err != nil {
		return model.MaintenanceProcedure{}, err, code
	}
	for _, procedure := range procedures {
		if procedure.PublicEventId == publicEventId {
			return procedure, nil, http.StatusOK
		}
	}
	return model.MaintenanceProcedure{}, errors.New("not found"), http.StatusNotFound
}

func (this *Controller) GetMaintenanceProcedureParametersOfInstance(token auth.Token, instanceId string, publicEventId string) ([]model.SmartServiceExtendedParameter, error, int) {
	procedure, err, code := this.GetMaintenanceProcedureOfInstance(token, instanceId, publicEventId)
	if err != nil {
		return nil, err, code
	}
	return this.parameterDescriptionsToSmartServiceExtendedParameter(token, procedure.ParameterDescriptions)
}

func (this *Controller) StartMaintenanceProcedure(token auth.Token, instanceId string, publicEventId string, parameters model.SmartServiceParameters) (error, int) {
	procedure, err, code := this.GetMaintenanceProcedureOfInstance(token, instanceId, publicEventId)
	if err != nil {
		return err, code
	}

	instance, err, code := this.GetInstance(token, instanceId)
	if err != nil {
		return err, code
	}

	parameterWithInstanceInputs := instance.Parameters
	parameterWithInstanceInputs = append(parameterWithInstanceInputs, parameters...)

	paramListWithAutoSelect, err, code := this.appendAutoSelectParams(token, parameterWithInstanceInputs, procedure.ParameterDescriptions)
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

	err = this.camunda.StartMaintenance(procedure, maintenanceId, paramListWithAutoSelect)
	if err != nil {
		return err, http.StatusInternalServerError
	}
	return nil, http.StatusOK
}
