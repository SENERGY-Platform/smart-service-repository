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

type ReleaseCommand struct {
	Command string                     `json:"command"`
	Id      string                     `json:"id"`
	Owner   string                     `json:"owner"`
	Release *model.SmartServiceRelease `json:"release"`
}

func (this *Controller) HandleReleaseMessage(delivery []byte) error {
	release := ReleaseCommand{}
	err := json.Unmarshal(delivery, &release)
	if err != nil {
		log.Println("ERROR: consumed invalid message --> ignore", err)
		debug.PrintStack()
		return err
	}
	return this.HandleRelease(release)
}

func (this *Controller) HandleRelease(cmd ReleaseCommand) (err error) {
	switch cmd.Command {
	case "PUT":
		if cmd.Release == nil {
			log.Println("WARNING: missing release in release put command", cmd)
			return nil
		}
		err = this.HandleReleaseSave(*cmd.Release)
		if err != nil {
			return err
		}
		return nil
	case "DELETE":
		err = this.HandleReleaseDelete(cmd.Id)
		if err != nil {
			return err
		}
		return nil
	default:
		return errors.New("unable to handle command: " + cmd.Command)
	}
}

func (this *Controller) HandleReleaseSave(release model.SmartServiceRelease) error {
	panic("not implemented") //TODO
}

func (this *Controller) HandleReleaseDelete(id string) error {
	panic("not implemented") //TODO
}

func (this *Controller) SaveRelease(token auth.Token, release model.SmartServiceRelease) error {
	if this.releasesProducer == nil {
		return errors.New("edit is disabled")
	}
	panic("not implemented") //TODO
}

func (this *Controller) DeleteRelease(token auth.Token, id string) error {
	if this.releasesProducer == nil {
		return errors.New("edit is disabled")
	}
	panic("not implemented") //TODO
}
