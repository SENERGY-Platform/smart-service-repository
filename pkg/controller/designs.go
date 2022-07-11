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
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"github.com/beevik/etree"
	"log"
	"net/http"
	"runtime/debug"
	"time"
)

func (this *Controller) ListDesigns(token auth.Token, query model.DesignQueryOptions) ([]model.SmartServiceDesign, error, int) {
	return this.db.ListDesigns(token.GetUserId(), query)
}

func (this *Controller) GetDesign(token auth.Token, id string) (result model.SmartServiceDesign, err error, code int) {
	return this.db.GetDesign(id, token.GetUserId())
}

func (this *Controller) SetDesign(token auth.Token, element model.SmartServiceDesign) (result model.SmartServiceDesign, err error, code int) {
	if element.Name == "" {
		element.Name, err = getProcessModelName(element.BpmnXml)
		if err != nil {
			return result, err, http.StatusBadRequest
		}
	}
	if element.Description == "" {
		element.Description, err = getProcessModelDescription(element.BpmnXml)
		if err != nil {
			return result, err, http.StatusBadRequest
		}
	}
	element.UpdatedAt = time.Now().Unix()
	err, code = this.ValidateDesign(token, element)
	if err != nil {
		return result, err, code
	}
	err, code = this.db.SetDesign(element)
	if err != nil {
		return result, err, code
	}
	return this.db.GetDesign(element.Id, token.GetUserId())
}

func (this *Controller) DeleteDesign(token auth.Token, id string) (error, int) {
	return this.db.DeleteDesign(id, token.GetUserId())
}

func (this *Controller) ValidateDesign(token auth.Token, element model.SmartServiceDesign) (err error, code int) {
	if element.Id == "" {
		return errors.New("missing id"), http.StatusBadRequest
	}
	if element.Name == "" {
		return errors.New("missing name"), http.StatusBadRequest
	}
	if element.UserId == "" {
		return errors.New("missing user id"), http.StatusBadRequest
	}
	if element.SvgXml == "" {
		return errors.New("missing svg xml"), http.StatusBadRequest
	}
	if element.BpmnXml == "" {
		return errors.New("missing bpmn xml"), http.StatusBadRequest
	}
	if err = validateBpmnXml(element.BpmnXml); err != nil {
		return err, http.StatusBadRequest
	}
	return nil, http.StatusOK
}

func validateBpmnXml(xml string) (err error) {
	if xml == "" {
		return errors.New("missing bpmn xml")
	}
	defer func() {
		if r := recover(); r != nil && err == nil {
			log.Printf("%s: %s", r, debug.Stack())
			err = errors.New(fmt.Sprint("Recovered Error: ", r))
		}
	}()
	doc := etree.NewDocument()
	err = doc.ReadFromString(xml)
	if err != nil {
		return err
	}
	definition := doc.FindElement("//bpmn:process")
	if definition == nil {
		return errors.New("missing process definition")
	}
	id := definition.SelectAttrValue("id", "")
	if id == "" {
		return errors.New("missing process definition id")
	}
	return nil
}

func getProcessModelName(bpmn string) (name string, err error) {
	defer func() {
		if r := recover(); r != nil && err == nil {
			log.Printf("%s: %s", r, debug.Stack())
			err = errors.New(fmt.Sprint("Recovered Error: ", r))
		}
	}()
	doc := etree.NewDocument()
	err = doc.ReadFromString(bpmn)
	if err != nil {
		return "", err
	}

	if len(doc.FindElements("//bpmn:collaboration")) > 0 {
		name = doc.FindElement("//bpmn:collaboration").SelectAttrValue("id", "process-name")
		name = doc.FindElement("//bpmn:collaboration").SelectAttrValue("name", name)
	} else {
		name = doc.FindElement("//bpmn:process").SelectAttrValue("id", "process-name")
		name = doc.FindElement("//bpmn:process").SelectAttrValue("name", name)
	}
	return name, nil
}

func getProcessModelDescription(bpmn string) (name string, err error) {
	defer func() {
		if r := recover(); r != nil && err == nil {
			log.Printf("%s: %s", r, debug.Stack())
			err = errors.New(fmt.Sprint("Recovered Error: ", r))
		}
	}()
	doc := etree.NewDocument()
	err = doc.ReadFromString(bpmn)
	if err != nil {
		return "", err
	}

	if len(doc.FindElements("//bpmn:collaboration")) > 0 {
		name = doc.FindElement("//bpmn:collaboration").SelectAttrValue("senergy:description", "")
	} else {
		name = doc.FindElement("//bpmn:process").SelectAttrValue("senergy:description", "")
	}
	return name, nil
}
