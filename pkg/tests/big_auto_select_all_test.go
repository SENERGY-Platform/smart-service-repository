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

package tests

import (
	"context"
	"encoding/json"
	"github.com/SENERGY-Platform/models/go/models"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/tests/mocks"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/tests/resources"
	"io"
	"net/http"
	"net/url"
	"runtime/debug"
	"sync"
	"testing"
	"time"
)

func TestIBigInstanceAutoSelectAllInput(t *testing.T) {
	t.Skip("no handling of big variables implemented")
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	apiUrl, config, testDeviceRepoDb, err := apiTestEnv(ctx, wg, true, resources.BigSelectionsResponseObj, func(err error) {
		debug.PrintStack()
		t.Error(err)
	})
	if err != nil {
		t.Error(err)
		return
	}
	err = testDeviceRepoDb.SetCharacteristic(ctx, models.Characteristic{
		Id:   "urn:infai:ses:characteristic:0b041ea3-8efd-4ce4-8130-d8af320326a4",
		Name: "location",
	})
	if err != nil {
		t.Error(err)
		return
	}

	topicBackup := mocks.CAMUNDA_MODULE_WORKER_TOPIC
	defer func() {
		mocks.CAMUNDA_MODULE_WORKER_TOPIC = topicBackup
	}()
	mocks.CAMUNDA_MODULE_WORKER_TOPIC = "info"

	mocks.NewModuleWorker(ctx, wg, apiUrl, config, func(taskWorkerMsg mocks.ModuleWorkerMessage) (err error) {
		actual, _ := json.Marshal(taskWorkerMsg.Variables)
		t.Log("worker call:", string(actual))
		return nil
	})

	design := model.SmartServiceDesign{}
	t.Run("create design", func(t *testing.T) {
		resp, err := post(userToken, apiUrl+"/designs", model.SmartServiceDesign{
			BpmnXml: resources.BigInputBpmn,
			SvgXml:  resources.ProcessDeploymentSvg,
		})
		if err != nil {
			t.Error(err)
			return
		}
		if resp.StatusCode != http.StatusOK {
			temp, _ := io.ReadAll(resp.Body)
			t.Error(resp.StatusCode, string(temp))
			return
		}
		checkContentType(t, resp)
		err = json.NewDecoder(resp.Body).Decode(&design)
		if err != nil {
			t.Error(err)
			return
		}
	})

	release := model.SmartServiceRelease{}
	t.Run("create release", func(t *testing.T) {
		resp, err := post(userToken, apiUrl+"/releases", model.SmartServiceRelease{
			DesignId:    design.Id,
			Name:        "release name",
			Description: "test description",
		})
		if err != nil {
			t.Error(err)
			return
		}
		if resp.StatusCode != http.StatusOK {
			temp, _ := io.ReadAll(resp.Body)
			t.Error(resp.StatusCode, string(temp))
			return
		}
		checkContentType(t, resp)
		err = json.NewDecoder(resp.Body).Decode(&release)
		if err != nil {
			t.Error(err)
			return
		}
	})

	time.Sleep(5 * time.Second) //allow async cqrs

	instance := model.SmartServiceInstance{}
	t.Run("create instance", func(t *testing.T) {
		resp, err := post(userToken, apiUrl+"/releases/"+url.PathEscape(release.Id)+"/instances", model.SmartServiceInstanceInit{
			SmartServiceInstanceInfo: model.SmartServiceInstanceInfo{
				Name:        "instance name",
				Description: "instance description",
			},
			Parameters: []model.SmartServiceParameter{
				{
					Id: "location",
					Value: map[string]interface{}{
						"Latitude":  51.338527718877394,
						"Longitude": 12.38074998525586,
					},
					Label:      "location",
					ValueLabel: "Augustusplatz, Leipzig",
				},
			},
		})
		if err != nil {
			t.Error(err)
			return
		}
		if resp.StatusCode != http.StatusOK {
			temp, _ := io.ReadAll(resp.Body)
			t.Error(resp.StatusCode, string(temp))
			return
		}
		checkContentType(t, resp)
		err = json.NewDecoder(resp.Body).Decode(&instance)
		if err != nil {
			t.Error(err)
			return
		}
	})

	time.Sleep(5 * time.Second)
	t.Log(instance)
}
