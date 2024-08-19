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
	devicerepository "github.com/SENERGY-Platform/device-repository/lib/client"
	permclient "github.com/SENERGY-Platform/permissions-v2/pkg/client"
	permmodel "github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"github.com/google/uuid"
	"sync"
)

type Controller struct {
	config            configuration.Config
	db                Database
	camunda           Camunda
	permissions       Permissions
	devicerepo        devicerepository.Interface
	selectables       Selectables
	userTokenProvider UserTokenProvider
	adminAccess       *auth.OpenidToken
	cleanupMux        sync.Mutex
}

type Permissions = permclient.Client

type Camunda interface {
	DeployRelease(owner string, release model.SmartServiceReleaseExtended) (err error, isInvalidCamundaDeployment bool)
	RemoveRelease(id string) error
	Start(result model.SmartServiceInstance) error
	CheckInstanceReady(smartServiceInstanceId string) (finished bool, missing bool, err error)
	StopInstance(smartServiceInstanceId string) error
	DeleteInstance(instance model.HistoricProcessInstance) (err error)
	GetProcessInstanceBusinessKey(processInstanceId string) (string, error, int)
	GetProcessInstanceList() (result []model.HistoricProcessInstance, err error)
	StartMaintenance(releaseId string, procedure model.MaintenanceProcedure, id string, parameter []model.SmartServiceParameter) error
}

type Selectables interface {
	Get(token auth.Token, searchedEntities []string, criteria []model.Criteria) (result []model.Selectable, err error, code int)
}

type UserTokenProvider = func(userid string) (token auth.Token, err error)

type Consumer = func(ctx context.Context, config configuration.Config, topic string, listener func(delivery []byte) error) error

func New(ctx context.Context, config configuration.Config, db Database, permissions Permissions, camunda Camunda, selectables Selectables, userTokenProvider UserTokenProvider, devicerepo devicerepository.Interface) (ctrl *Controller, err error) {
	ctrl = &Controller{
		config:            config,
		db:                db,
		permissions:       permissions,
		camunda:           camunda,
		selectables:       selectables,
		userTokenProvider: userTokenProvider,
		adminAccess:       &auth.OpenidToken{},
		devicerepo:        devicerepo,
	}
	_, err, _ = permissions.SetTopic(permclient.InternalAdminToken, permclient.Topic{
		Id: config.SmartServiceReleasePermissionsTopic,
		DefaultPermissions: permmodel.ResourcePermissions{
			RolePermissions: map[string]permmodel.PermissionsMap{
				"admin": {
					Read:         true,
					Write:        true,
					Execute:      true,
					Administrate: true,
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return ctrl, nil
}

func (this *Controller) GetNewId() string {
	return uuid.NewString()
}
