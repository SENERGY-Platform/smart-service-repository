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
	"github.com/SENERGY-Platform/smart-service-repository/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/database/mongo"
	"github.com/google/uuid"
)

type Controller struct {
	db               Database
	releasesProducer Producer
}

type Producer interface {
	Produce(key string, message []byte) error
}

type GenericProducerFactory[T Producer] func(ctx context.Context, config configuration.Config, topic string) (T, error)
type ProducerFactory = GenericProducerFactory[Producer]
type Consumer = func(ctx context.Context, config configuration.Config, topic string, listener func(delivery []byte) error) error

func New(ctx context.Context, config configuration.Config, db *mongo.Mongo, consumer Consumer, producer ProducerFactory) (ctrl *Controller, err error) {
	ctrl = &Controller{
		db: db,
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
