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
	"log"
	"net/http"
	"net/url"
	"runtime/debug"
)

func (this *Camunda) Start(instance model.SmartServiceInstance) error {
	requestBody := new(bytes.Buffer)
	query, err := createCamundaStartForm(instance)
	if err != nil {
		return err
	}
	err = json.NewEncoder(requestBody).Encode(query)
	if err != nil {
		return err
	}
	key := idToCNName(instance.ReleaseId)
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

		//TODO remove debug read
		debugRead, _ := this.getProcessDefinitionList()
		temp, _ := json.Marshal(debugRead)
		log.Println("definitions:", string(temp))
		return err
	}
	return nil
}

func createCamundaStartForm(instance model.SmartServiceInstance) (result CamundaStartForm, err error) {
	result.BusinessKey = instance.Id
	result.Variables = map[string]CamundaStartVariable{}
	for _, param := range instance.Parameters {
		result.Variables[param.Id] = CamundaStartVariable{Value: param.Value}
	}
	return result, nil
}

type CamundaStartForm struct {
	BusinessKey string                          `json:"businessKey"`
	Variables   map[string]CamundaStartVariable `json:"variables"`
}

type CamundaStartVariable struct {
	Value interface{} `json:"value"`
}
