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
	"github.com/SENERGY-Platform/smart-service-repository/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/database/mongo"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/permissions"
	"github.com/google/uuid"
)

type Controller struct {
	config            configuration.Config
	db                Database
	camunda           Camunda
	releasesProducer  Producer
	permissions       Permissions
	selectables       Selectables
	userTokenProvider UserTokenProvider
}

type Producer interface {
	Produce(key string, message []byte) error
}

type Permissions interface {
	CheckAccess(token auth.Token, topic string, id string, right string) (bool, error)
	Query(token string, query permissions.QueryMessage, result interface{}) (err error, code int)
}

type Camunda interface {
	DeployRelease(owner string, release model.SmartServiceReleaseExtended) (err error, isInvalidCamundaDeployment bool)
	RemoveRelease(id string) error
	Start(result model.SmartServiceInstance) error
	CheckInstanceReady(smartServiceInstanceId string) (finished bool, missing bool)
	StopInstance(smartServiceInstanceId string) error
	DeleteInstance(instance model.HistoricProcessInstance) (err error)
	GetProcessInstanceBusinessKey(processInstanceId string) (string, error, int)
	GetProcessInstanceList() (result []model.HistoricProcessInstance, err error)
}

type Selectables interface {
	Get(token auth.Token, searchedEntities []string, criteria []model.Criteria) (result []model.Selectable, err error, code int)
}

type UserTokenProvider = func(userid string) (token auth.Token, err error)

type GenericProducerFactory[T Producer] func(ctx context.Context, config configuration.Config, topic string) (T, error)
type ProducerFactory = GenericProducerFactory[Producer]
type Consumer = func(ctx context.Context, config configuration.Config, topic string, listener func(delivery []byte) error) error

func New(ctx context.Context, config configuration.Config, db *mongo.Mongo, permissions Permissions, camunda Camunda, selectables Selectables, consumer Consumer, producer ProducerFactory, userTokenProvider UserTokenProvider) (ctrl *Controller, err error) {
	ctrl = &Controller{
		config:            config,
		db:                db,
		permissions:       permissions,
		camunda:           camunda,
		selectables:       selectables,
		userTokenProvider: userTokenProvider,
	}
	if config.EditForward == "" || config.EditForward == "-" {
		ctrl.releasesProducer, err = producer(ctx, config, config.KafkaSmartServiceReleaseTopic)
		if err != nil {
			return ctrl, err
		}
		err = consumer(ctx, config, config.KafkaSmartServiceReleaseTopic, ctrl.HandleReleaseMessage)
		if err != nil {
			return ctrl, err
		}
	}
	return ctrl, nil
}

func (this *Controller) GetNewId() string {
	return uuid.NewString()
}
