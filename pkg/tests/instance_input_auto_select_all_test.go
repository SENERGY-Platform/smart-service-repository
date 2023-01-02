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
	"github.com/SENERGY-Platform/smart-service-repository/pkg/kafka"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/tests/mocks"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/tests/resources"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"runtime/debug"
	"sync"
	"testing"
	"time"
)

func TestInstanceAutoSelectAllInput(t *testing.T) {
	if CI {
		t.Skip("not in ci")
	}
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	apiUrl, config, err := apiTestEnv(ctx, wg, true, resources.SelectionsResponse4Obj, func(err error) {
		debug.PrintStack()
		t.Error(err)
	})
	if err != nil {
		t.Error(err)
		return
	}

	characteristicsProducer, err := kafka.NewProducer(ctx, config, config.KafkaCharacteristicsTopic)
	if err != nil {
		t.Error(err)
		return
	}
	characteristicMsg, err := json.Marshal(map[string]interface{}{
		"command": "PUT",
		"id":      "urn:infai:ses:characteristic:0b041ea3-8efd-4ce4-8130-d8af320326a4",
		"owner":   userId,
		"characteristic": model.Characteristic{
			Id:   "urn:infai:ses:characteristic:0b041ea3-8efd-4ce4-8130-d8af320326a4",
			Name: "location",
		},
	})
	if err != nil {
		t.Error(err)
		return
	}
	err = characteristicsProducer.Produce("location", characteristicMsg)
	if err != nil {
		t.Error(err)
		return
	}
	time.Sleep(10 * time.Second)

	topicBackup := mocks.CAMUNDA_MODULE_WORKER_TOPIC
	defer func() {
		mocks.CAMUNDA_MODULE_WORKER_TOPIC = topicBackup
	}()
	mocks.CAMUNDA_MODULE_WORKER_TOPIC = "info"

	mocks.NewModuleWorker(ctx, wg, apiUrl, config, func(taskWorkerMsg mocks.ModuleWorkerMessage) (err error) {
		expectedVariables := map[string]mocks.CamundaVariable{
			"info.module_data": {
				Type:  "String",
				Value: "{\n                        \"lat\": \"51.338527718877394\",\n                        \"lon\": \"12.38074998525586\",\n                        \"location\": \"{\"Latitude\":51.338527718877394,\"Longitude\":12.38074998525586}\"\n                        }",
			},
			"lat": {
				Type:  "Double",
				Value: 51.338527718877394,
			},
			"lon": {
				Type:  "Double",
				Value: 12.38074998525586,
			},
			"location": {
				Type:  "String",
				Value: "{\"Latitude\":51.338527718877394,\"Longitude\":12.38074998525586}",
			},
			"devices": {
				Type:  "String",
				Value: "[\"{\\\"device_selection\\\":{\\\"device_id\\\":\\\"device_1\\\",\\\"service_id\\\":null,\\\"path\\\":null},\\\"label\\\":\\\"Device 1: one service, no paths\\\"}\",\"{\\\"device_selection\\\":{\\\"device_id\\\":\\\"device_2\\\",\\\"service_id\\\":null,\\\"path\\\":null},\\\"label\\\":\\\"Device 2: one service, one path\\\"}\",\"{\\\"device_selection\\\":{\\\"device_id\\\":\\\"device_3\\\",\\\"service_id\\\":null,\\\"path\\\":null},\\\"label\\\":\\\"Device 3: one service, two paths\\\"}\",\"{\\\"device_selection\\\":{\\\"device_id\\\":\\\"device_4\\\",\\\"service_id\\\":null,\\\"path\\\":null},\\\"label\\\":\\\"Device 4: two services, no paths\\\"}\",\"{\\\"device_selection\\\":{\\\"device_id\\\":\\\"device_5\\\",\\\"service_id\\\":null,\\\"path\\\":null},\\\"label\\\":\\\"Device 5: two services, one path\\\"}\",\"{\\\"device_selection\\\":{\\\"device_id\\\":\\\"device_6\\\",\\\"service_id\\\":null,\\\"path\\\":null},\\\"label\\\":\\\"Device 6: two services, two paths\\\"}\",\"{\\\"device_selection\\\":{\\\"device_id\\\":\\\"device_7\\\",\\\"service_id\\\":null,\\\"path\\\":null},\\\"label\\\":\\\"Device 7: 3 services, mixed paths arrangements\\\"}\"]",
			},
		}
		actual, _ := json.Marshal(taskWorkerMsg.Variables)
		expected, _ := json.Marshal(expectedVariables)
		t.Log("worker call:", string(actual))
		if !reflect.DeepEqual(taskWorkerMsg.Variables, expectedVariables) {
			t.Error("\n", string(expected), "\n", string(actual))
		}
		return nil
	})

	design := model.SmartServiceDesign{}
	t.Run("create design", func(t *testing.T) {
		resp, err := post(userToken, apiUrl+"/designs", model.SmartServiceDesign{
			BpmnXml: resources.AutoSelectAllInputBpmn,
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
