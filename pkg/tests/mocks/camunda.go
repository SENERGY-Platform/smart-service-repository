/*
 * Copyright 2024 InfAI (CC SES)
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

package mocks

import (
	"context"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
)

type CamundaErrMock struct {
	Err error
}

func (this *CamundaErrMock) DeployRelease(_ context.Context, owner string, release model.SmartServiceReleaseExtended) (err error, isInvalidCamundaDeployment bool) {
	return this.Err, false
}

func (this *CamundaErrMock) RemoveRelease(_ context.Context, id string) error {
	return this.Err
}

func (this *CamundaErrMock) Start(_ context.Context, result model.SmartServiceInstance) error {
	return this.Err
}

func (this *CamundaErrMock) CheckInstanceReady(_ context.Context, smartServiceInstanceId string) (finished bool, missing bool, err error) {
	//TODO implement me
	panic("implement me")
}

func (this *CamundaErrMock) StopInstance(_ context.Context, smartServiceInstanceId string) error {
	//TODO implement me
	panic("implement me")
}

func (this *CamundaErrMock) DeleteInstance(_ context.Context, instance model.HistoricProcessInstance) (err error) {
	//TODO implement me
	panic("implement me")
}

func (this *CamundaErrMock) GetProcessInstanceBusinessKey(_ context.Context, processInstanceId string) (string, error, int) {
	//TODO implement me
	panic("implement me")
}

func (this *CamundaErrMock) GetProcessInstanceList(_ context.Context) (result []model.HistoricProcessInstance, err error) {
	//TODO implement me
	panic("implement me")
}

func (this *CamundaErrMock) StartMaintenance(_ context.Context, releaseId string, procedure model.MaintenanceProcedure, id string, parameter []model.SmartServiceParameter) error {
	//TODO implement me
	panic("implement me")
}
