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
	"encoding/json"
	"errors"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"log"
	"runtime/debug"
)

func (this *Controller) HandleDeploymentMessage(delivery []byte) error {
	deployment := DeploymentCommand{}
	err := json.Unmarshal(delivery, &deployment)
	if err != nil {
		log.Println("ERROR: consumed invalid message --> ignore", err)
		debug.PrintStack()
		return err
	}
	return this.HandleDeployment(deployment)
}

func (this *Controller) HandleDeployment(cmd DeploymentCommand) (err error) {
	switch cmd.Command {
	case "PUT":
		if cmd.Deployment == nil {
			log.Println("WARNING: missing deployment in deployment put command", cmd)
			return nil
		}
		err = this.HandleDeploymentSave(*cmd.Deployment)
		if err != nil {
			return err
		}
		return nil
	case "DELETE":
		err = this.HandleDeploymentDelete(cmd.Id)
		if err != nil {
			return err
		}
		return nil
	default:
		return errors.New("unable to handle command: " + cmd.Command)
	}
}

func (this *Controller) HandleDeploymentSave(deployment model.SmartServiceDeployment) error {
	panic("not implemented") //TODO
}

func (this *Controller) HandleDeploymentDelete(id string) error {
	panic("not implemented") //TODO
}

func (this *Controller) SaveDeployment(token auth.Token, deployment model.SmartServiceDeployment) error {
	if this.deploymentsProducer == nil {
		return errors.New("edit is disabled")
	}
	panic("not implemented") //TODO
}

func (this *Controller) DeleteDeployment(token auth.Token, id string) error {
	if this.deploymentsProducer == nil {
		return errors.New("edit is disabled")
	}
	panic("not implemented") //TODO
}
