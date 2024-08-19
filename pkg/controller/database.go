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

import "github.com/SENERGY-Platform/smart-service-repository/pkg/model"

type Database interface {
	DesignsInterface
	ModuleInterface
	InstanceInterface
	ReleaseInterface
	MaintenanceInterface
	VariableInterface
}

type DesignsInterface interface {
	GetDesign(id string, userId string) (model.SmartServiceDesign, error, int)
	SetDesign(element model.SmartServiceDesign) (error, int)
	DeleteDesign(id string, userId string) (error, int)
	ListDesigns(userId string, query model.DesignQueryOptions) ([]model.SmartServiceDesign, error, int)
}

type ModuleInterface interface {
	SetModule(element model.SmartServiceModule) (error, int)
	SetModules(element []model.SmartServiceModule) (error, int)
	GetModule(id string, userId string) (model.SmartServiceModule, error, int)
	DeleteModule(id string, userId string) (error, int)
	ListModules(userId string, query model.ModuleQueryOptions) ([]model.SmartServiceModule, error, int)
	ListAllModules(query model.ModuleQueryOptions) (result []model.SmartServiceModule, err error, code int)
	SetInstanceError(id string, userId string, errMsg string) error
}

type InstanceInterface interface {
	GetInstance(id string, userId string) (model.SmartServiceInstance, error, int)
	DeleteInstance(id string, userId string) (error, int)
	SetInstance(element model.SmartServiceInstance) (error, int)
	ListInstances(userId string, query model.InstanceQueryOptions) (result []model.SmartServiceInstance, err error, code int)
	ListInstancesOfRelease(userId string, releaseId string) (result []model.SmartServiceInstance, err error, code int)
}

type ReleaseInterface interface {
	SetRelease(element model.SmartServiceReleaseExtended, markAsDone bool) (error, int)
	MarkReleaseAsFinished(id string) (err error)

	GetRelease(id string, withMarked bool) (model.SmartServiceReleaseExtended, error, int)
	ListReleases(options model.ListReleasesOptions) ([]model.SmartServiceReleaseExtended, error)
	GetReleasesByDesignId(designId string) ([]model.SmartServiceReleaseExtended, error)

	MarlReleaseAsDeleted(id string) (error, int)
	DeleteRelease(id string) (error, int)

	GetMarkedReleases() (markedAsDeleted []model.SmartServiceReleaseExtended, markedAsUnfinished []model.SmartServiceReleaseExtended, err error)
}

type MaintenanceInterface interface {
	RemoveFromRunningMaintenanceIds(instanceId string, removeMaintenanceIds []string) error
	AddToRunningMaintenanceIds(instanceId string, maintenanceId string) error
}

type VariableInterface interface {
	GetVariable(instanceId string, userId string, variableName string) (result model.SmartServiceInstanceVariable, err error, code int)
	SetVariable(element model.SmartServiceInstanceVariable) (model.SmartServiceInstanceVariable, error, int)
	DeleteVariable(instanceId string, userId string, variableName string) (error, int)
	ListVariables(instanceId string, userId string, query model.VariableQueryOptions) (result []model.SmartServiceInstanceVariable, err error, code int)
	ListAllVariables(query model.VariableQueryOptions) (result []model.SmartServiceInstanceVariable, err error, code int)
}
