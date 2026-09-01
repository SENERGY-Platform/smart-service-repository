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
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime/debug"

	"github.com/SENERGY-Platform/gin-middleware/otelx"

	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
)

func (this *Camunda) StopInstance(ctx context.Context, smartServiceInstanceId string) error {
	instances, err := this.getProcessInstanceListByKey(ctx, smartServiceInstanceId)
	if err != nil {
		return err
	}
	for _, instance := range instances {
		err = this.DeleteInstance(ctx, instance)
		if err != nil {
			return err
		}
	}
	return nil
}

func (this *Camunda) DeleteInstance(ctx context.Context, instance model.HistoricProcessInstance) (err error) {
	if instance.EndTime == "" {
		err = this.deleteInstance(ctx, instance.Id)
		if err != nil {
			return err
		}
	}
	err = this.deleteInstanceHistory(ctx, instance.Id)
	if err != nil {
		return err
	}
	return nil
}

func (this *Camunda) deleteInstance(ctx context.Context, id string) (err error) {
	req, err := http.NewRequest("DELETE", this.config.CamundaUrl+"/engine-rest/process-instance/"+url.PathEscape(id)+"?skipIoMappings=true&failIfNotExists=false", nil)
	if err != nil {
		return this.filterUrlFromErr(err)
	}
	err = otelx.InjectContextToRequest(ctx, req)
	if err != nil {
		return this.filterUrlFromErr(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		err = this.filterUrlFromErr(err)
		this.config.GetLogger().ErrorContext(ctx, "error in deleteInstance", "error", err, "stack", string(debug.Stack()))
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		temp, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unable to delete process-instance: %v, %v", resp.StatusCode, string(temp))
	}
	return nil
}

func (this *Camunda) deleteInstanceHistory(ctx context.Context, id string) (err error) {
	req, err := http.NewRequest("DELETE", this.config.CamundaUrl+"/engine-rest/history/process-instance/"+url.PathEscape(id)+"?failIfNotExists=false", nil)
	if err != nil {
		return this.filterUrlFromErr(err)
	}
	err = otelx.InjectContextToRequest(ctx, req)
	if err != nil {
		return this.filterUrlFromErr(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		err = this.filterUrlFromErr(err)
		this.config.GetLogger().ErrorContext(ctx, "error in deleteInstanceHistory", "error", err, "stack", string(debug.Stack()))
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		temp, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unable to delete process-instance: %v, %v", resp.StatusCode, string(temp))
	}
	return nil
}

func (this *Camunda) getProcessInstanceListByKey(ctx context.Context, key string) (result []model.HistoricProcessInstance, err error) {
	req, err := http.NewRequest("GET", this.config.CamundaUrl+"/engine-rest/history/process-instance?processInstanceBusinessKey="+url.QueryEscape(key), nil)
	if err != nil {
		return result, this.filterUrlFromErr(err)
	}
	err = otelx.InjectContextToRequest(ctx, req)
	if err != nil {
		return result, this.filterUrlFromErr(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		err = this.filterUrlFromErr(err)
		this.config.GetLogger().ErrorContext(ctx, "error in getProcessInstanceListByKey", "error", err, "stack", string(debug.Stack()))
		return result, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		temp, _ := io.ReadAll(resp.Body)
		return result, fmt.Errorf("unable to get process-instance list by key: %v, %v", resp.StatusCode, string(temp))
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	return
}
