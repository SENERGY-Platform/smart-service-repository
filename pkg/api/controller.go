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
	"github.com/SENERGY-Platform/smart-service-repository/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
)

type Controller interface {
	DesignsInterface
	ModulesInterface
	ReleaseInterface
	GetNewId() string
}

type ModulesInterface interface {
	AddModule(token auth.Token, processInstanceId string, module model.SmartServiceModuleInit) (model.SmartServiceModule, error, int)
	ListModules(token auth.Token, query model.ModuleQueryOptions) ([]model.SmartServiceModule, error, int)
}

type DesignsInterface interface {
	ListDesigns(token auth.Token, query model.DesignQueryOptions) ([]model.SmartServiceDesign, error, int)
	GetDesign(token auth.Token, id string) (model.SmartServiceDesign, error, int)
	SetDesign(token auth.Token, element model.SmartServiceDesign) (model.SmartServiceDesign, error, int)
	DeleteDesign(token auth.Token, id string) (error, int)
}

type ReleaseInterface interface {
	CreateRelease(token auth.Token, element model.SmartServiceRelease) (model.SmartServiceRelease, error, int)
	DeleteRelease(token auth.Token, id string) (error, int)
	GetRelease(token auth.Token, id string) (model.SmartServiceRelease, error, int)
	ListReleases(token auth.Token, query model.ReleaseQueryOptions) ([]model.SmartServiceRelease, error, int)
	GetReleaseParameter(token auth.Token, id string) ([]model.SmartServiceExtendedParameter, error, int)
}
