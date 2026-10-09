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

package api

import (
	"context"

	"github.com/SENERGY-Platform/smart-service-repository/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
)

type Controller interface {
	DesignsInterface
	ModulesInterface
	BulkModulesInterface
	ReleaseInterface
	InstancesInterface
	MaintenanceInterface
	VariablesInterface
	GetNewId() string
}

type ModulesInterface interface {
	SetModuleForProcessInstance(ctx context.Context, processInstanceId string, module model.SmartServiceModuleInit, moduleId string) (model.SmartServiceModule, error, int)
	AddModuleForProcessInstance(ctx context.Context, processInstanceId string, module model.SmartServiceModuleInit) (model.SmartServiceModule, error, int)
	ListModulesOfProcessInstance(ctx context.Context, processInstanceId string, query model.ModuleQueryOptions) ([]model.SmartServiceModule, error, int)
	AddModule(ctx context.Context, token auth.Token, instanceId string, module model.SmartServiceModuleInit) (model.SmartServiceModule, error, int)
	ListModules(ctx context.Context, token auth.Token, query model.ModuleQueryOptions) ([]model.SmartServiceModule, error, int)
	DeleteModule(ctx context.Context, token auth.Token, id string, ignoreModuleDeleteError bool) (error, int)
	GetModule(ctx context.Context, token auth.Token, id string) (model.SmartServiceModule, error, int)
	SetModuleError(ctx context.Context, token auth.Token, moduleId string, errMsg string) (error, int)
}

type BulkModulesInterface interface {
	AddModulesForProcessInstance(ctx context.Context, processInstanceId string, module []model.SmartServiceModuleInit) ([]model.SmartServiceModule, error, int)
}

type DesignsInterface interface {
	ListDesigns(ctx context.Context, token auth.Token, query model.DesignQueryOptions) ([]model.SmartServiceDesign, error, int)
	GetDesign(ctx context.Context, token auth.Token, id string) (model.SmartServiceDesign, error, int)
	SetDesign(ctx context.Context, token auth.Token, element model.SmartServiceDesign) (model.SmartServiceDesign, error, int)
	DeleteDesign(ctx context.Context, token auth.Token, id string) (error, int)
}

type ReleaseInterface interface {
	CreateRelease(ctx context.Context, token auth.Token, element model.SmartServiceRelease) (model.SmartServiceRelease, error, int)
	DeleteRelease(ctx context.Context, token auth.Token, id string, deletePreviousReleases bool) (error, int)
	GetRelease(ctx context.Context, token auth.Token, id string) (model.SmartServiceRelease, error, int)
	GetExtendedRelease(ctx context.Context, token auth.Token, id string) (model.SmartServiceReleaseExtended, error, int)
	ListReleases(ctx context.Context, token auth.Token, query model.ReleaseQueryOptions) ([]model.SmartServiceRelease, int64, error, int)
	ListExtendedReleases(ctx context.Context, token auth.Token, query model.ReleaseQueryOptions) (result []model.SmartServiceReleaseExtended, total int64, err error, code int)
	GetReleaseParameter(ctx context.Context, token auth.Token, id string) ([]model.SmartServiceExtendedParameter, error, int)
	GetReleaseParameterWithoutAuthCheck(ctx context.Context, token auth.Token, id string) (result []model.SmartServiceExtendedParameter, err error, code int)
	GetReleaseInstanceCount(ctx context.Context, token auth.Token, id string) (count int64, err error, code int)
}

type InstancesInterface interface {
	CreateInstance(ctx context.Context, token auth.Token, releaseId string, instance model.SmartServiceInstanceInit) (model.SmartServiceInstance, error, int)
	ListInstances(ctx context.Context, token auth.Token, query model.InstanceQueryOptions) ([]model.SmartServiceInstance, int64, error, int)
	GetInstance(ctx context.Context, token auth.Token, id string) (model.SmartServiceInstance, error, int)
	DeleteInstance(ctx context.Context, token auth.Token, id string, ignoreModuleDeleteError bool) (error, int)
	SetInstanceError(ctx context.Context, token auth.Token, instanceId string, errMsg string) (error, int)
	SetInstanceErrorByProcessInstanceId(ctx context.Context, processInstanceId string, errMsg string) (error, int)
	UpdateInstanceInfo(ctx context.Context, token auth.Token, id string, element model.SmartServiceInstanceInfo) (model.SmartServiceInstance, error, int)
	RedeployInstance(ctx context.Context, token auth.Token, id string, parameters []model.SmartServiceParameter, releaseId string) (model.SmartServiceInstance, error, int)
	GetInstanceUserIdByProcessInstanceId(ctx context.Context, processInstanceId string) (string, error, int)
	GetInstanceByProcessInstanceId(ctx context.Context, processInstanceId string) (model.SmartServiceInstance, error, int)
}

type MaintenanceInterface interface {
	GetMaintenanceProceduresOfInstance(ctx context.Context, token auth.Token, instanceId string) (maintenanceProcedure []model.MaintenanceProcedure, instance model.SmartServiceInstance, release model.SmartServiceReleaseExtended, err error, code int)
	GetMaintenanceProcedureOfInstance(ctx context.Context, token auth.Token, instanceId string, publicEventId string) (maintenanceProcedure model.MaintenanceProcedure, instance model.SmartServiceInstance, release model.SmartServiceReleaseExtended, err error, code int)
	GetMaintenanceProcedureParametersOfInstance(ctx context.Context, token auth.Token, instanceId string, publicEventId string) ([]model.SmartServiceExtendedParameter, error, int)
	StartMaintenanceProcedure(ctx context.Context, token auth.Token, instanceId string, publicEventId string, parameters model.SmartServiceParameters) (error, int)
}

type VariablesInterface interface {
	SetVariable(ctx context.Context, token auth.Token, variable model.SmartServiceInstanceVariable) (result model.SmartServiceInstanceVariable, err error, code int)
	GetVariablesMap(ctx context.Context, token auth.Token, instanceId string, query model.VariableQueryOptions) (map[string]interface{}, error, int)
	ListVariables(ctx context.Context, token auth.Token, instanceId string, query model.VariableQueryOptions) ([]model.SmartServiceInstanceVariable, error, int)
	DeleteVariable(ctx context.Context, token auth.Token, instanceId string, name string) (error, int)
	GetVariable(ctx context.Context, token auth.Token, instanceId string, name string) (model.SmartServiceInstanceVariable, error, int)
	SetVariablesMapOfProcessInstance(ctx context.Context, processInstanceId string, mappedVariableValues map[string]interface{}) (err error, code int)
	GetVariablesMapOfProcessInstance(ctx context.Context, processInstanceId string) (map[string]interface{}, error, int)
}
