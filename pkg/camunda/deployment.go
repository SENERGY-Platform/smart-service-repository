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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/SENERGY-Platform/gin-middleware/otelx"

	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/notification"
	"github.com/beevik/etree"
)

func (this *Camunda) DeployRelease(ctx context.Context, owner string, release model.SmartServiceReleaseExtended) (err error, isInvalidCamundaDeployment bool) {
	id := idToCNName(release.Id)
	err = this.RemoveRelease(ctx, release.Id) //remove existing releases with the same id
	if err != nil {
		return err, false
	}
	releaseXml, err := this.modifyBpmnWithReleaseIds(ctx, release.BpmnXml, id, release.ParsedInfo.MaintenanceProcedures)
	if err != nil {
		return err, true
	}
	responseWrapper, err, code := this.deployProcess(ctx, release.Name, releaseXml, release.SvgXml)
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
			_ = notification.Send(ctx, this.config.NotificationUrl, notification.Message{
				UserId:  owner,
				Title:   "Smart-Service-Release Error: ProcessEngineException",
				Message: msg,
			}, this.config.GetLogger())
			return fmt.Errorf("unable to release: %v", msg), true
		}
		return errors.New("unknown release error"), true
	}
	return nil, false
}

func (this *Camunda) deployProcess(ctx context.Context, name string, xml string, svg string) (result map[string]interface{}, err error, code int) {
	result = map[string]interface{}{}
	boundary := "---------------------------" + time.Now().String()
	b := strings.NewReader(buildDeploymentPayLoad(name, xml, svg, boundary))
	req, err := http.NewRequest("POST", this.config.CamundaUrl+"/engine-rest/deployment/create", b)
	if err != nil {
		err = this.filterUrlFromErr(err)
		this.config.GetLogger().ErrorContext(ctx, "error in request to processengine ", "error", err, "stack", string(debug.Stack()))
		return result, err, 0
	}
	req.Header.Set("Content-Type", "multipart/form-data; boundary="+boundary)
	err = otelx.InjectContextToRequest(ctx, req)
	if err != nil {
		return result, err, 0
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		err = this.filterUrlFromErr(err)
		this.config.GetLogger().ErrorContext(ctx, "error in request to processengine ", "error", err, "stack", string(debug.Stack()))
		return result, err, 0
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&result)
	return result, err, resp.StatusCode
}

func (this *Camunda) modifyBpmnWithReleaseIds(ctx context.Context, xml string, id string, maintenanceProcedures []model.MaintenanceProcedure) (resultXml string, err error) {
	defer func() {
		if r := recover(); r != nil && err == nil {
			this.config.GetLogger().ErrorContext(ctx, "error in modifyBpmnWithReleaseIds", "error", r, "stack", string(debug.Stack()))
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

	for _, maintenance := range maintenanceProcedures {
		ref := doc.FindElement("//bpmn:message[@id='" + maintenance.MessageRef + "']")
		if ref == nil {
			return "", fmt.Errorf("unknown maintenance message ref %v", maintenance.MessageRef)
		}
		ref.CreateAttr("name", maintenance.InternalEventId)
	}

	return doc.WriteToString()
}

func buildDeploymentPayLoad(name string, xml string, svg string, boundary string) string {
	segments := []string{
		"Content-Disposition: form-data; name=\"data\"; " + "filename=\"" + name + ".bpmn\"\r\nContent-Type: text/xml\r\n\r\n" + xml + "\r\n",
		"Content-Disposition: form-data; name=\"diagram\"; " + "filename=\"" + name + ".svg\"\r\nContent-Type: image/svg+xml\r\n\r\n" + svg + "\r\n",
		"Content-Disposition: form-data; name=\"deployment-name\"\r\n\r\n" + name + "\r\n",
	}
	return "--" + boundary + "\r\n" + strings.Join(segments, "--"+boundary+"\r\n") + "--" + boundary + "--\r\n"
}
