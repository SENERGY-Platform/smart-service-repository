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

	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
)

type Database interface {
	DesignsInterface
	ModuleInterface
	InstanceInterface
	ReleaseInterface
	MaintenanceInterface
	VariableInterface
}

type DesignsInterface interface {
	GetDesign(ctx context.Context, id string, userId string) (model.SmartServiceDesign, error, int)
	SetDesign(ctx context.Context, element model.SmartServiceDesign) (error, int)
	DeleteDesign(ctx context.Context, id string, userId string) (error, int)
	ListDesigns(ctx context.Context, userId string, query model.DesignQueryOptions) ([]model.SmartServiceDesign, error, int)
}

type ModuleInterface interface {
	SetModule(ctx context.Context, element model.SmartServiceModule) (error, int)
	SetModules(ctx context.Context, element []model.SmartServiceModule) (error, int)
	GetModule(ctx context.Context, id string, userId string) (model.SmartServiceModule, error, int)
	DeleteModule(ctx context.Context, id string, userId string) (error, int)
	ListModules(ctx context.Context, userId string, query model.ModuleQueryOptions) ([]model.SmartServiceModule, error, int)
	ListAllModules(ctx context.Context, query model.ModuleQueryOptions) (result []model.SmartServiceModule, err error, code int)
	SetInstanceError(ctx context.Context, id string, userId string, errMsg string) error
	SetModuleError(ctx context.Context, id string, userId string, errMsg string) error
}

type InstanceInterface interface {
	GetInstance(ctx context.Context, id string, userId string) (model.SmartServiceInstance, error, int)
	DeleteInstance(ctx context.Context, id string, userId string) (error, int)
	SetInstance(ctx context.Context, element model.SmartServiceInstance) (error, int)
	ListInstances(ctx context.Context, userId string, query model.InstanceQueryOptions) (result []model.SmartServiceInstance, total int64, err error, code int)
	ListInstancesOfRelease(ctx context.Context, userId string, releaseId string) (result []model.SmartServiceInstance, err error, code int)
	CountInstancesOfRelease(ctx context.Context, releaseId string) (count int64, err error, code int)
	CountInstancesOfReleases(ctx context.Context, releaseIds []string) (counts map[string]int64, err error)
}

type ReleaseInterface interface {
	SetRelease(ctx context.Context, element model.SmartServiceReleaseExtended, markAsUnfinished bool) (error, int)
	MarkReleaseAsFinished(ctx context.Context, id string) (err error)

	GetRelease(ctx context.Context, id string, withMarked bool) (model.SmartServiceReleaseExtended, error, int)
	ListReleases(ctx context.Context, options model.ListReleasesOptions) ([]model.SmartServiceReleaseExtended, int64, error)
	GetReleasesByDesignId(ctx context.Context, designId string) ([]model.SmartServiceReleaseExtended, error)
	GetPreviousReleases(ctx context.Context, releaseId string) (result []model.SmartServiceReleaseExtended, err error)

	MarlReleaseAsDeleted(ctx context.Context, id string) (error, int)
	DeleteRelease(ctx context.Context, id string) (error, int)

	GetMarkedReleases(ctx context.Context) (markedAsDeleted []model.SmartServiceReleaseExtended, markedAsUnfinished []model.SmartServiceReleaseExtended, err error)

	ForEachReleaseWithoutUsedResources(ctx context.Context, f func(release model.SmartServiceReleaseExtended) error) error
	SetReleaseUsedResources(ctx context.Context, id string, used model.ReleaseUsedResources) (updated bool, err error)
	ListReleasesUsingResource(ctx context.Context, kind model.ResourceKind, resourceId string) ([]model.SmartServiceRelease, error)
	ListFinishedReleaseIds(ctx context.Context, ids []string) ([]string, error)
}

type MaintenanceInterface interface {
	RemoveFromRunningMaintenanceIds(ctx context.Context, instanceId string, removeMaintenanceIds []string) error
	AddToRunningMaintenanceIds(ctx context.Context, instanceId string, maintenanceId string) error
}

type VariableInterface interface {
	GetVariable(ctx context.Context, instanceId string, userId string, variableName string) (result model.SmartServiceInstanceVariable, err error, code int)
	SetVariable(ctx context.Context, element model.SmartServiceInstanceVariable) (model.SmartServiceInstanceVariable, error, int)
	DeleteVariable(ctx context.Context, instanceId string, userId string, variableName string) (error, int)
	ListVariables(ctx context.Context, instanceId string, userId string, query model.VariableQueryOptions) (result []model.SmartServiceInstanceVariable, err error, code int)
	ListAllVariables(ctx context.Context, query model.VariableQueryOptions) (result []model.SmartServiceInstanceVariable, err error, code int)
}
