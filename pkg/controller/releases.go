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
	"fmt"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"log"
	"net/http"
	"runtime/debug"
	"time"
)

//---------- API -------------

func (this *Controller) CreateRelease(token auth.Token, element model.SmartServiceRelease) (result model.SmartServiceRelease, err error, code int) {
	if this.releasesProducer == nil {
		return result, errors.New("edit is disabled"), http.StatusInternalServerError
	}
	if element.DesignId == "" {
		return result, errors.New("missing design id"), http.StatusBadRequest
	}
	design, err, code := this.GetDesign(token, element.DesignId)
	if err != nil {
		if code == http.StatusNotFound {
			return result, fmt.Errorf("user does not own a smart-service-design with the id %v", element.DesignId), http.StatusBadRequest
		}
		return result, err, http.StatusInternalServerError
	}
	if element.Name == "" {
		element.Name = design.Name
	}
	if element.Description == "" {
		element.Description = design.Description
	}
	element.CreatedAt = time.Now().Unix()

	if element.Id == "" {
		element.Id = this.GetNewId()
	}

	parsedInfo, err := this.parseDesignXmlForReleaseInfo(design.BpmnXml)
	if err != nil {
		return result, fmt.Errorf("unable to parse design xml for release: %w", err), http.StatusBadRequest
	}

	msg, err := json.Marshal(ReleaseCommand{
		Command: "PUT",
		Id:      element.Id,
		Owner:   token.GetUserId(),
		Release: &model.SmartServiceReleaseExtended{
			SmartServiceRelease: element,
			BpmnXml:             design.BpmnXml,
			SvgXml:              design.SvgXml,
			ParsedInfo:          parsedInfo,
		},
	})
	if err != nil {
		return result, err, http.StatusInternalServerError
	}

	err = this.releasesProducer.Produce(element.Id, msg)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}

	return element, nil, http.StatusOK
}

func (this *Controller) GetRelease(token auth.Token, id string) (result model.SmartServiceRelease, err error, code int) {
	access, err := this.permissions.CheckAccess(token, this.config.KafkaSmartServiceReleaseTopic, id, "r")
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	if !access {
		return result, errors.New("access denied"), http.StatusForbidden
	}
	var extended model.SmartServiceReleaseExtended
	extended, err, code = this.db.GetRelease(id)
	return extended.SmartServiceRelease, err, code
}

func (this *Controller) DeleteRelease(token auth.Token, id string) (error, int) {
	if this.releasesProducer == nil {
		return errors.New("edit is disabled"), http.StatusInternalServerError
	}
	access, err := this.permissions.CheckAccess(token, this.config.KafkaSmartServiceReleaseTopic, id, "a")
	if err != nil {
		return err, http.StatusInternalServerError
	}
	if !access {
		return errors.New("access denied"), http.StatusForbidden
	}
	msg, err := json.Marshal(ReleaseCommand{
		Command: "DELETE",
		Id:      id,
	})
	if err != nil {
		return err, http.StatusInternalServerError
	}
	err = this.releasesProducer.Produce(id, msg)
	if err != nil {
		return err, http.StatusInternalServerError
	}
	return nil, http.StatusOK
}

//---------- CQRS -------------

type ReleaseCommand struct {
	Command string                             `json:"command"`
	Id      string                             `json:"id"`
	Owner   string                             `json:"owner"`
	Release *model.SmartServiceReleaseExtended `json:"release"`
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
		err = this.HandleReleaseSave(cmd.Owner, *cmd.Release)
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

func (this *Controller) HandleReleaseSave(owner string, release model.SmartServiceReleaseExtended) error {
	err, _ := this.db.SetRelease(release)
	if err != nil {
		return err
	}
	err, isInvalidCamundaDeployment := this.camunda.DeployRelease(owner, release)
	if err != nil {
		if isInvalidCamundaDeployment {
			return this.db.SetReleaseError(release.Id, err.Error())
		} else {
			return err
		}
	}
	return nil
}

func (this *Controller) HandleReleaseDelete(id string) error {
	err := this.camunda.RemoveRelease(id)
	if err != nil {
		return err
	}
	err, _ = this.db.DeleteRelease(id)
	return err
}

//------------ Parsing ----------------

func (this *Controller) parseDesignXmlForReleaseInfo(xml string) (model.SmartServiceReleaseInfo, error) {
	//TODO
	return model.SmartServiceReleaseInfo{}, nil
}
