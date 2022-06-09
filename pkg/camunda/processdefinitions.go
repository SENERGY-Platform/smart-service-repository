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
	"io"
	"log"
	"net/http"
	"net/url"
	"runtime/debug"
)

func (this *Camunda) getProcessDefinition(id string) (result ProcessDefinition, exists bool, err error) {
	key := idToCNName(id)
	req, err := http.NewRequest("GET", this.config.CamundaUrl+"/engine-rest/process-definition/key/"+url.PathEscape(key), nil)
	if err != nil {
		err = this.filterUrlFromErr(err)
		log.Println("ERROR:", err)
		debug.PrintStack()
		return result, false, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		err = this.filterUrlFromErr(err)
		log.Println("ERROR:", err)
		debug.PrintStack()
		return result, false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return result, false, nil
	}
	if resp.StatusCode >= 300 {
		temp, _ := io.ReadAll(resp.Body)
		err = errors.New(string(temp))
		err = this.filterUrlFromErr(err)
		log.Println("ERROR:", err)
		debug.PrintStack()
		return result, false, err
	}
	exists = true
	err = json.NewDecoder(resp.Body).Decode(&result)
	return
}

func (this *Camunda) getProcessDefinitionList() (result []ProcessDefinition, err error) {
	req, err := http.NewRequest("GET", this.config.CamundaUrl+"/engine-rest/process-definition", nil)
	if err != nil {
		return result, this.filterUrlFromErr(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		err = this.filterUrlFromErr(err)
		debug.PrintStack()
		return result, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		temp, _ := io.ReadAll(resp.Body)
		return result, fmt.Errorf("unable to get process-definition list: %v, %v", resp.StatusCode, string(temp))
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	return
}

func (this *Camunda) getProcessDefinitionListByKey(key string) (result []ProcessDefinition, err error) {
	req, err := http.NewRequest("GET", this.config.CamundaUrl+"/engine-rest/process-definition?key="+url.QueryEscape(key), nil)
	if err != nil {
		return result, this.filterUrlFromErr(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		err = this.filterUrlFromErr(err)
		debug.PrintStack()
		log.Println("ERROR:", err)
		return result, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		temp, _ := io.ReadAll(resp.Body)
		return result, fmt.Errorf("unable to get process-definition list by key: %v, %v", resp.StatusCode, string(temp))
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	return
}
