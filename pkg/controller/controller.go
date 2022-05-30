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
	"github.com/SENERGY-Platform/smart-service-repository/pkg/kafka"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
)

type Controller struct {
	deploymentsProducer *kafka.Producer
}

type DeploymentCommand struct {
	Command    string                     `json:"command"`
	Id         string                     `json:"id"`
	Owner      string                     `json:"owner"`
	Deployment *model.SmartServiceRelease `json:"deployment"`
}

func New(ctx context.Context, config configuration.Config) (ctrl *Controller, err error) {
	ctrl = &Controller{}
	if config.EditForward == "" || config.EditForward == "-" {
		ctrl.deploymentsProducer, err = kafka.NewProducer(ctx, config, config.KafkaSmartServiceDeploymentTopic)
		if err != nil {
			return ctrl, err
		}
		err = kafka.NewConsumer(ctx, config, config.KafkaSmartServiceDeploymentTopic, ctrl.HandleDeploymentMessage)
		if err != nil {
			return ctrl, err
		}
	}
	return ctrl, nil
}
