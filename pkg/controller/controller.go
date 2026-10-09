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
	"maps"
	"slices"
	"sync"

	devicerepository "github.com/SENERGY-Platform/device-repository/v2/lib/client"
	permclient "github.com/SENERGY-Platform/permissions-v2/pkg/client"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"github.com/google/uuid"
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

	usedResourcesBackfillMux sync.Mutex
}

type Permissions = permclient.Client

type Camunda interface {
	DeployRelease(ctx context.Context, owner string, release model.SmartServiceReleaseExtended) (err error, isInvalidCamundaDeployment bool)
	RemoveRelease(ctx context.Context, id string) error
	Start(ctx context.Context, result model.SmartServiceInstance) error
	CheckInstanceReady(ctx context.Context, smartServiceInstanceId string) (finished bool, missing bool, err error)
	StopInstance(ctx context.Context, smartServiceInstanceId string) error
	DeleteInstance(ctx context.Context, instance model.HistoricProcessInstance) (err error)
	GetProcessInstanceBusinessKey(ctx context.Context, processInstanceId string) (string, error, int)
	GetProcessInstanceList(ctx context.Context) (result []model.HistoricProcessInstance, err error)
	StartMaintenance(ctx context.Context, releaseId string, procedure model.MaintenanceProcedure, id string, parameter []model.SmartServiceParameter) error
}

type Selectables interface {
	Get(ctx context.Context, token auth.Token, searchedEntities []string, criteria []model.Criteria) (result []model.Selectable, err error, code int)
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
	topicDesc := configuration.GetTopicDesc(config)
	_, err, _ = permissions.SetTopicContext(ctx, permclient.InternalAdminToken, topicDesc)
	if err != nil {
		return nil, err
	}

	topicDesc.Id = config.SmartServiceInstancePermissionsTopic
	_, err, _ = permissions.SetTopicContext(ctx, permclient.InternalAdminToken, topicDesc)
	if err != nil {
		return nil, err
	}

	instances, _, err, _ := db.ListInstances(ctx, "", model.InstanceQueryOptions{})
	if err != nil {
		return nil, err
	}

	permResources, err, _ := permissions.ListResourcesWithAdminPermissionContext(ctx, permclient.InternalAdminToken, config.SmartServiceInstancePermissionsTopic, permclient.ListOptions{})
	if err != nil {
		return nil, err
	}
	permResouceMap := map[string]permclient.Resource{}
	for _, permResource := range permResources {
		permResouceMap[permResource.Id] = permResource
	}

	dbIds := []string{}
	for _, instance := range instances {
		dbIds = append(dbIds, instance.Id)

		_, ok := permResouceMap[instance.Id]
		if !ok {
			// missing in permissions v2
			perm := permclient.ResourcePermissions{
				UserPermissions: map[string]permclient.PermissionsMap{
					instance.UserId: permclient.PermissionsMap{
						Administrate: true,
						Write:        true,
						Read:         true,
						Execute:      true,
					},
				},
			}
			_, err, _ = permissions.SetPermissionContext(ctx, permclient.InternalAdminToken, config.SmartServiceInstancePermissionsTopic, instance.Id, perm)
			if err != nil {
				return nil, err
			}
		}
	}

	permResouceIds := maps.Keys(permResouceMap)

	for permResouceId := range permResouceIds {
		if !slices.Contains(dbIds, permResouceId) {
			err, _ = permissions.RemoveResourceContext(ctx, permclient.InternalAdminToken, config.SmartServiceInstancePermissionsTopic, permResouceId)
			if err != nil {
				return nil, err
			}
		}
	}

	return ctrl, nil
}

func (this *Controller) GetNewId() string {
	return uuid.NewString()
}
