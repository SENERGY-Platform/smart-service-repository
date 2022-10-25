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
	"bytes"
	"encoding/json"
	"errors"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"runtime/debug"
	"strings"
)

func (this *Camunda) Start(instance model.SmartServiceInstance) error {
	requestBody := new(bytes.Buffer)
	key := idToCNName(instance.ReleaseId)
	variables, err := this.GetProcessParameters(key)
	if err != nil {
		return err
	}
	query, err := createCamundaStartForm(instance, variables)
	if err != nil {
		return err
	}
	err = json.NewEncoder(requestBody).Encode(query)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", this.config.CamundaUrl+"/engine-rest/process-definition/key/"+url.PathEscape(key)+"/submit-form", requestBody)
	if err != nil {
		debug.PrintStack()
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		debug.PrintStack()
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		err = errors.New(buf.String())
		log.Println("ERROR: ", resp.StatusCode, err)
		debug.PrintStack()
		return err
	}
	_, _ = io.ReadAll(resp.Body)
	return nil
}

func (this *Camunda) StartMaintenance(releaseId string, procedure model.MaintenanceProcedure, id string, parameter []model.SmartServiceParameter) error {
	requestBody := new(bytes.Buffer)
	key := idToCNName(releaseId)
	variables, err := this.GetProcessParameters(key)
	if err != nil {
		return err
	}
	query, err := createMaintenanceStartForm(procedure, id, parameter, variables)
	if err != nil {
		return err
	}
	err = json.NewEncoder(requestBody).Encode(query)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", this.config.CamundaUrl+"/engine-rest/message", requestBody)
	if err != nil {
		debug.PrintStack()
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		debug.PrintStack()
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		err = errors.New(buf.String())
		log.Println("ERROR: ", resp.StatusCode, err)
		debug.PrintStack()
		return err
	}
	_, _ = io.ReadAll(resp.Body)
	return nil
}

type Variable struct {
	Value     interface{} `json:"value"`
	Type      string      `json:"type"`
	ValueInfo interface{} `json:"valueInfo"`
}

func (this *Camunda) GetProcessParameters(processDefinitionKey string) (result map[string]Variable, err error) {
	req, err := http.NewRequest("GET", this.config.CamundaUrl+"/engine-rest/process-definition/key/"+url.PathEscape(processDefinitionKey)+"/form-variables", nil)
	if err != nil {
		return result, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		temp, _ := ioutil.ReadAll(resp.Body)
		err = errors.New(resp.Status + " " + string(temp))
		return
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	return
}

func createCamundaStartForm(instance model.SmartServiceInstance, variables map[string]Variable) (result CamundaStartForm, err error) {
	result.BusinessKey = instance.Id
	result.Variables = map[string]CamundaStartVariable{
		//model.CamundaUserIdParameter: {Value: instance.UserId},
	}
	for _, param := range instance.Parameters {
		value, err := handleObjectsAsJson(param, variables)
		if err != nil {
			return result, err
		}
		result.Variables[param.Id] = CamundaStartVariable{Value: value}
	}
	return result, nil
}

func createMaintenanceStartForm(procedure model.MaintenanceProcedure, id string, parameter []model.SmartServiceParameter, variables map[string]Variable) (result CamundaMaintenanceStartForm, err error) {
	result.BusinessKey = id
	result.MessageName = procedure.InternalEventId
	result.ProcessVariables = map[string]CamundaStartVariable{}
	for _, param := range parameter {
		value, err := handleObjectsAsJson(param, variables)
		if err != nil {
			return result, err
		}
		result.ProcessVariables[param.Id] = CamundaStartVariable{Value: value}
	}
	return result, nil
}

func handleObjectsAsJson(param model.SmartServiceParameter, variables map[string]Variable) (result interface{}, err error) {
	if variable, ok := variables[param.Id]; ok {
		typeStr := strings.ToLower(variable.Type)
		if typeStr == "string" || typeStr == "text" {
			if _, isStr := param.Value.(string); !isStr {
				temp, err := json.Marshal(param.Value)
				if err != nil {
					return result, err
				}
				return string(temp), nil
			}
		}
	}
	return param.Value, nil
}

type CamundaStartForm struct {
	BusinessKey string                          `json:"businessKey"`
	Variables   map[string]CamundaStartVariable `json:"variables"`
}

type CamundaMaintenanceStartForm struct {
	MessageName      string                          `json:"messageName"`
	BusinessKey      string                          `json:"businessKey"`
	ProcessVariables map[string]CamundaStartVariable `json:"processVariables"`
}

type CamundaStartVariable struct {
	Value interface{} `json:"value"`
}
