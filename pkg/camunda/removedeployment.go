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
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime/debug"
)

func (this *Camunda) RemoveRelease(id string) error {
	id = idToCNName(id)
	deplIds, err := this.getDeploymentIds(id)
	if err != nil {
		return err
	}
	if len(deplIds) > 0 {
		this.config.GetLogger().Debug("remove deployments", "ids", deplIds)
	}
	for _, deplId := range deplIds {
		err = this.removeDeployment(deplId)
		if err != nil {
			return fmt.Errorf("unable to delete release %v\n%w", id, err)
		}
	}
	return nil
}

func (this *Camunda) removeDeployment(deplId string) error {
	req, err := http.NewRequest("DELETE", this.config.CamundaUrl+"/engine-rest/deployment/"+url.PathEscape(deplId)+"?cascade=true&skipIoMappings=true", nil)
	if err != nil {
		err = this.filterUrlFromErr(err)
		this.config.GetLogger().Error("error in removeDeployment", "error", err, "stack", string(debug.Stack()))
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		err = this.filterUrlFromErr(err)
		this.config.GetLogger().Error("error in removeDeployment", "error", err, "stack", string(debug.Stack()))
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		temp, _ := io.ReadAll(resp.Body)
		err = fmt.Errorf("unable to remove deployment (%v) from camunda: %v", deplId, string(temp))
		this.config.GetLogger().Error("error in removeDeployment", "error", err, "stack", string(debug.Stack()))
		return err
	}
	_, _ = io.ReadAll(resp.Body)
	return nil
}

func (this *Camunda) getDeploymentId(id string) (deplId string, exists bool, err error) {
	var definition ProcessDefinition
	definition, exists, err = this.getProcessDefinition(id)
	return definition.DeploymentId, exists, err
}

func (this *Camunda) getDeploymentIds(id string) (deplIds []string, err error) {
	var definitions []ProcessDefinition
	definitions, err = this.getProcessDefinitionListByKey(id)
	if err != nil {
		return deplIds, err
	}
	for _, definition := range definitions {
		deplIds = append(deplIds, definition.DeploymentId)
	}
	return deplIds, nil
}
