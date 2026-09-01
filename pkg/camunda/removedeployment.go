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
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime/debug"

	"github.com/SENERGY-Platform/gin-middleware/otelx"
)

func (this *Camunda) RemoveRelease(ctx context.Context, id string) error {
	id = idToCNName(id)
	deplIds, err := this.getDeploymentIds(ctx, id)
	if err != nil {
		return err
	}
	if len(deplIds) > 0 {
		this.config.GetLogger().DebugContext(ctx, "remove deployments", "ids", deplIds)
	}
	for _, deplId := range deplIds {
		err = this.removeDeployment(ctx, deplId)
		if err != nil {
			return fmt.Errorf("unable to delete release %v\n%w", id, err)
		}
	}
	return nil
}

func (this *Camunda) removeDeployment(ctx context.Context, deplId string) error {
	req, err := http.NewRequest("DELETE", this.config.CamundaUrl+"/engine-rest/deployment/"+url.PathEscape(deplId)+"?cascade=true&skipIoMappings=true", nil)
	if err != nil {
		err = this.filterUrlFromErr(err)
		this.config.GetLogger().ErrorContext(ctx, "error in removeDeployment", "error", err, "stack", string(debug.Stack()))
		return err
	}
	err = otelx.InjectContextToRequest(ctx, req)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		err = this.filterUrlFromErr(err)
		this.config.GetLogger().ErrorContext(ctx, "error in removeDeployment", "error", err, "stack", string(debug.Stack()))
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		temp, _ := io.ReadAll(resp.Body)
		err = fmt.Errorf("unable to remove deployment (%v) from camunda: %v", deplId, string(temp))
		this.config.GetLogger().ErrorContext(ctx, "error in removeDeployment", "error", err, "stack", string(debug.Stack()))
		return err
	}
	_, _ = io.ReadAll(resp.Body)
	return nil
}

func (this *Camunda) getDeploymentId(ctx context.Context, id string) (deplId string, exists bool, err error) {
	var definition ProcessDefinition
	definition, exists, err = this.getProcessDefinition(ctx, id)
	return definition.DeploymentId, exists, err
}

func (this *Camunda) getDeploymentIds(ctx context.Context, id string) (deplIds []string, err error) {
	var definitions []ProcessDefinition
	definitions, err = this.getProcessDefinitionListByKey(ctx, id)
	if err != nil {
		return deplIds, err
	}
	for _, definition := range definitions {
		deplIds = append(deplIds, definition.DeploymentId)
	}
	return deplIds, nil
}
