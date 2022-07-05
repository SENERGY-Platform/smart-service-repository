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
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"io"
	"log"
	"net/http"
	"net/url"
	"runtime/debug"
)

func (this *Camunda) StopInstance(smartServiceInstanceId string) error {
	instances, err := this.getProcessInstanceListByKey(smartServiceInstanceId)
	if err != nil {
		return err
	}
	for _, instance := range instances {
		err = this.DeleteInstance(instance)
		if err != nil {
			return err
		}
	}
	return nil
}

func (this *Camunda) DeleteInstance(instance model.HistoricProcessInstance) (err error) {
	if instance.EndTime == "" {
		err = this.deleteInstance(instance.Id)
		if err != nil {
			return err
		}
	}
	err = this.deleteInstanceHistory(instance.Id)
	if err != nil {
		return err
	}
	return nil
}

func (this *Camunda) deleteInstance(id string) (err error) {
	req, err := http.NewRequest("DELETE", this.config.CamundaUrl+"/engine-rest/process-instance/"+url.PathEscape(id)+"?skipIoMappings=true", nil)
	if err != nil {
		return this.filterUrlFromErr(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		err = this.filterUrlFromErr(err)
		debug.PrintStack()
		log.Println("ERROR:", err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 && resp.StatusCode != 404 {
		temp, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unable to delete process-instance: %v, %v", resp.StatusCode, string(temp))
	}
	return nil
}

func (this *Camunda) deleteInstanceHistory(id string) (err error) {
	req, err := http.NewRequest("DELETE", this.config.CamundaUrl+"/engine-rest/history/process-instance/"+url.PathEscape(id), nil)
	if err != nil {
		return this.filterUrlFromErr(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		err = this.filterUrlFromErr(err)
		debug.PrintStack()
		log.Println("ERROR:", err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 && resp.StatusCode != 404 {
		temp, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unable to delete process-instance: %v, %v", resp.StatusCode, string(temp))
	}
	return nil
}

func (this *Camunda) getProcessInstanceListByKey(key string) (result []model.HistoricProcessInstance, err error) {
	req, err := http.NewRequest("GET", this.config.CamundaUrl+"/engine-rest/history/process-instance?processInstanceBusinessKey="+url.QueryEscape(key), nil)
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
		return result, fmt.Errorf("unable to get process-instance list by key: %v, %v", resp.StatusCode, string(temp))
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	return
}
