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

package camunda

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/notification"
	"github.com/beevik/etree"
	"io"
	"log"
	"net/http"
	"net/url"
	"runtime/debug"
	"strings"
	"time"
)

type Camunda struct {
	config configuration.Config
}

func New(config configuration.Config) *Camunda {
	return &Camunda{
		config: config,
	}
}

func (this *Camunda) RemoveRelease(id string) error {
	deplId, exists, err := this.getDeploymentId(id)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	req, err := http.NewRequest("DELETE", this.config.CamundaUrl+"/deployment/"+url.PathEscape(deplId), nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		temp, _ := io.ReadAll(resp.Body)
		err = fmt.Errorf("unable to remove release (%v, %v) from camunda: %v", id, deplId, string(temp))
		return err
	}
	return nil
}

func (this *Camunda) DeployRelease(owner string, release model.SmartServiceReleaseExtended) (err error, isInvalidCamundaDeployment bool) {
	err = this.RemoveRelease(release.Id) //remove existing releases with the same id
	if err != nil {
		return err, false
	}
	releaseXml, err := setReleaseIdToBpmn(release.BpmnXml, release.Id)
	if err != nil {
		return err, true
	}
	responseWrapper, err, code := this.deployProcess(release.Name, releaseXml, release.SvgXml)
	if err != nil {
		return err, code > 0
	}
	ok := false
	_, ok = responseWrapper["id"].(string)
	if !ok {
		if responseWrapper["type"] == "ProcessEngineException" {
			msg, ok := responseWrapper["message"].(string)
			if !ok {
				msg = "unknown error"
			}
			_ = notification.Send(this.config.NotificationUrl, notification.Message{
				UserId:  owner,
				Title:   "Smart-Service-Release Error: ProcessEngineException",
				Message: msg,
			})
			return fmt.Errorf("unable to release: %v", msg), true
		}
	}
	return errors.New("unknown release error"), true
}

func (this *Camunda) deployProcess(name string, xml string, svg string) (result map[string]interface{}, err error, code int) {
	result = map[string]interface{}{}
	boundary := "---------------------------" + time.Now().String()
	b := strings.NewReader(buildPayLoad(name, xml, svg, boundary))
	resp, err := http.Post(this.config.CamundaUrl+"/engine-rest/deployment/create", "multipart/form-data; boundary="+boundary, b)
	if err != nil {
		log.Println("ERROR: request to processengine ", err)
		return result, err, 0
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&result)
	return result, err, resp.StatusCode
}

func (this *Camunda) getDeploymentId(id string) (deplId string, exists bool, err error) {
	var definition ProcessDefinition
	definition, exists, err = this.getProcessDefinition(id)
	return definition.DeploymentId, exists, err
}

func (this *Camunda) getProcessDefinition(id string) (result ProcessDefinition, exists bool, err error) {
	req, err := http.NewRequest("GET", this.config.CamundaUrl+"/process-definition/key/"+url.PathEscape(id), nil)
	if err != nil {
		return result, false, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return result, false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return result, false, nil
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	return
}

func setReleaseIdToBpmn(xml string, id string) (resultXml string, err error) {
	defer func() {
		if r := recover(); r != nil && err == nil {
			log.Printf("%s: %s", r, debug.Stack())
			err = errors.New(fmt.Sprint("Recovered Error: ", r))
		}
	}()
	doc := etree.NewDocument()
	err = doc.ReadFromString(xml)
	if err != nil {
		return "", err
	}
	if len(doc.FindElements("//bpmn:collaboration")) > 0 {
		doc.FindElement("//bpmn:collaboration").CreateAttr("id", id)
	} else {
		doc.FindElement("//bpmn:process").CreateAttr("id", id)
	}
	return doc.WriteToString()
}

func buildPayLoad(name string, xml string, svg string, boundary string) string {
	segments := []string{
		"Content-Disposition: form-data; name=\"data\"; " + "filename=\"" + name + ".bpmn\"\r\nContent-Type: text/xml\r\n\r\n" + xml + "\r\n",
		"Content-Disposition: form-data; name=\"diagram\"; " + "filename=\"" + name + ".svg\"\r\nContent-Type: image/svg+xml\r\n\r\n" + svg + "\r\n",
		"Content-Disposition: form-data; name=\"deployment-name\"\r\n\r\n" + name + "\r\n",
	}
	return "--" + boundary + "\r\n" + strings.Join(segments, "--"+boundary+"\r\n") + "--" + boundary + "--\r\n"
}
