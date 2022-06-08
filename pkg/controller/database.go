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
}

type DesignsInterface interface {
	GetDesign(id string, userId string) (model.SmartServiceDesign, error, int)
	SetDesign(element model.SmartServiceDesign) (error, int)
	DeleteDesign(id string, userId string) (error, int)
	ListDesigns(userId string, query model.DesignQueryOptions) ([]model.SmartServiceDesign, error, int)
}

type ModuleInterface interface {
	SetModule(element model.SmartServiceModule) (error, int)
	GetModule(id string, userId string) (model.SmartServiceModule, error, int)
	ListModules(id string, query model.ModuleQueryOptions) ([]model.SmartServiceModule, error, int)
}

type InstanceInterface interface {
	GetInstance(id string, userId string) (model.SmartServiceInstance, error, int)
	DeleteInstance(id string, userId string) (error, int)
	SetInstance(element model.SmartServiceInstance) (error, int)
	ListInstances(userId string, query model.InstanceQueryOptions) (result []model.SmartServiceInstance, err error, code int)
}

type ReleaseInterface interface {
	SetRelease(element model.SmartServiceReleaseExtended) (error, int)
	SetReleaseError(id string, errMsg string) error
	GetRelease(id string) (model.SmartServiceReleaseExtended, error, int)
	GetReleaseListByIds(ids []string, sort string) ([]model.SmartServiceReleaseExtended, error)
	DeleteRelease(id string) (error, int)
}
