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
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime/debug"

	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
)

func (this *Camunda) GetProcessInstanceBusinessKey(processInstanceId string) (string, error, int) {
	instance, err := this.getProcessInstanceHistory(processInstanceId)
	if err != nil {
		return "", err, http.StatusInternalServerError
	}
	return instance.BusinessKey, nil, http.StatusOK
}

func (this *Camunda) getProcessInstanceHistory(processInstanceId string) (result HistoricProcessInstance, err error) {
	req, err := http.NewRequest("GET", this.config.CamundaUrl+"/engine-rest/history/process-instance/"+url.QueryEscape(processInstanceId), nil)
	if err != nil {
		return result, this.filterUrlFromErr(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		err = this.filterUrlFromErr(err)
		this.config.GetLogger().Error("error in getProcessInstanceHistory", "error", err, "stack", string(debug.Stack()))
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

func (this *Camunda) GetProcessInstanceList() (result []model.HistoricProcessInstance, err error) {
	req, err := http.NewRequest("GET", this.config.CamundaUrl+"/engine-rest/history/process-instance", nil)
	if err != nil {
		return result, this.filterUrlFromErr(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		err = this.filterUrlFromErr(err)
		this.config.GetLogger().Error("error in GetProcessInstanceList", "error", err, "stack", string(debug.Stack()))
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
