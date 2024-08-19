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
	"reflect"
	"runtime/debug"
	"sync"
	"testing"
	"time"
)

func TestMaintenanceProcedure(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	apiUrl, config, testDeviceRepoDb, err := apiTestEnv(ctx, wg, true, nil, func(err error) {
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

	isMaintenanceCall := false

	mocks.NewModuleWorker(ctx, wg, apiUrl, config, func(taskWorkerMsg mocks.ModuleWorkerMessage) (err error) {
		expectedVariables := map[string]mocks.CamundaVariable{
			"info.module_data": {
				Type:  "String",
				Value: "{\n\n                        }",
			},
			"info.module_type": {
				Type:  "String",
				Value: "widget",
			},
			"value_input": {
				Type:  "String",
				Value: "foo",
			},
			"iot_input": {
				Type:  "String",
				Value: "{\"device_selection\":{\"device_id\":\"device_1\",\"service_id\":\"s1\",\"path\":null},\"label\":\"Device 1: one service, no paths\"}",
			},
			"auto_input": {
				Type:  "String",
				Value: "[\"{\\\"device_selection\\\":{\\\"device_id\\\":\\\"device_1\\\",\\\"service_id\\\":null,\\\"path\\\":null},\\\"label\\\":\\\"Device 1: one service, no paths\\\"}\",\"{\\\"device_selection\\\":{\\\"device_id\\\":\\\"device_2\\\",\\\"service_id\\\":null,\\\"path\\\":null},\\\"label\\\":\\\"Device 2: one service, one path\\\"}\",\"{\\\"device_selection\\\":{\\\"device_id\\\":\\\"device_3\\\",\\\"service_id\\\":null,\\\"path\\\":null},\\\"label\\\":\\\"Device 3: one service, two paths\\\"}\",\"{\\\"device_selection\\\":{\\\"device_id\\\":\\\"device_4\\\",\\\"service_id\\\":null,\\\"path\\\":null},\\\"label\\\":\\\"Device 4: two services, no paths\\\"}\",\"{\\\"device_selection\\\":{\\\"device_id\\\":\\\"device_5\\\",\\\"service_id\\\":null,\\\"path\\\":null},\\\"label\\\":\\\"Device 5: two services, one path\\\"}\",\"{\\\"device_selection\\\":{\\\"device_id\\\":\\\"device_6\\\",\\\"service_id\\\":null,\\\"path\\\":null},\\\"label\\\":\\\"Device 6: two services, two paths\\\"}\",\"{\\\"device_selection\\\":{\\\"device_id\\\":\\\"device_7\\\",\\\"service_id\\\":null,\\\"path\\\":null},\\\"label\\\":\\\"Device 7: 3 services, mixed paths arrangements\\\"}\"]",
			},
		}
		if isMaintenanceCall {
			expectedVariables["pipeline_id"] = mocks.CamundaVariable{
				Type:  "String",
				Value: "foobar",
			}
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
			BpmnXml: resources.MaintenanceTestBpmn,
			SvgXml:  resources.MaintenanceTestSvg,
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

	extendedRelease := model.SmartServiceReleaseExtended{}
	t.Run("read extended release", func(t *testing.T) {
		resp, err := get(userToken, apiUrl+"/extended-releases/"+url.PathEscape(release.Id))
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
		err = json.NewDecoder(resp.Body).Decode(&extendedRelease)
		if err != nil {
			t.Error(err)
			return
		}
		expectedInfo := model.SmartServiceReleaseInfo{
			ParameterDescriptions: []model.ParameterDescription{
				{
					Id:           "value_input",
					Label:        "value_input",
					Type:         "string",
					DefaultValue: "",
				},
				{
					Id:           "iot_input",
					Label:        "iot_input",
					Type:         "string",
					DefaultValue: "",
					IotDescription: &model.IotDescription{
						TypeFilter:                   []model.FilterPossibility{"device"},
						Criteria:                     []model.Criteria{},
						EntityOnly:                   false,
						NeedsSameEntityIdInParameter: "",
					},
					Order: 0,
				},
				{
					Id:            "auto_input",
					Label:         "auto_input",
					Type:          "string",
					DefaultValue:  "",
					AutoSelectAll: true,
					Multiple:      true,
					IotDescription: &model.IotDescription{
						TypeFilter:                   []model.FilterPossibility{"device"},
						Criteria:                     []model.Criteria{},
						EntityOnly:                   true,
						NeedsSameEntityIdInParameter: "",
					},
					Order: 0,
				},
			},
			MaintenanceProcedures: []model.MaintenanceProcedure{
				{
					BpmnId:          "StartEvent_0at1j8v",
					MessageRef:      "Message_0ufoygk",
					PublicEventId:   "update",
					InternalEventId: release.Id + "_update",
					ParameterDescriptions: []model.ParameterDescription{
						{
							Id:           "pipeline_id",
							Label:        "pipeline_id",
							Type:         "string",
							DefaultValue: nil,
						},
					},
				},
			},
		}
		if !reflect.DeepEqual(extendedRelease.ParsedInfo, expectedInfo) {
			actual, _ := json.Marshal(extendedRelease.ParsedInfo)
			expected, _ := json.Marshal(expectedInfo)
			t.Error("\n", string(expected), "\n", string(actual))
		}
	})

	instance := model.SmartServiceInstance{}
	t.Run("create instance", func(t *testing.T) {
		resp, err := post(userToken, apiUrl+"/releases/"+url.PathEscape(release.Id)+"/instances", model.SmartServiceInstanceInit{
			SmartServiceInstanceInfo: model.SmartServiceInstanceInfo{
				Name:        "instance name",
				Description: "instance description",
			},
			Parameters: []model.SmartServiceParameter{
				{
					Id:         "value_input",
					Value:      "foo",
					Label:      "value_input",
					ValueLabel: "foo",
				},
				{
					Id:         "iot_input",
					Value:      "{\"device_selection\":{\"device_id\":\"device_1\",\"service_id\":\"s1\",\"path\":null},\"label\":\"Device 1: one service, no paths\"}",
					Label:      "iot_input",
					ValueLabel: "device_1",
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

	t.Run("read instance ready", func(t *testing.T) {
		resp, err := get(userToken, apiUrl+"/instances/"+url.PathEscape(instance.Id))
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

		if instance.Name != "instance name" {
			t.Error(instance.Name)
			return
		}
		if instance.Description != "instance description" {
			t.Error(instance.Description)
			return
		}
		if instance.UserId != userId {
			t.Error(instance.UserId)
			return
		}
		if instance.DesignId != design.Id {
			t.Error(instance.DesignId)
			return
		}
		if instance.ReleaseId != release.Id {
			t.Error(instance.ReleaseId)
			return
		}
		if !instance.Ready {
			t.Error(instance.Ready)
			return
		}
		if instance.Error != "" {
			t.Error(instance.Error)
			return
		}
	})
	t.Log(instance)

	isMaintenanceCall = true

	parameters := []model.SmartServiceExtendedParameter{}
	t.Run("read maintenance params", func(t *testing.T) {
		resp, err := get(userToken, apiUrl+"/instances/"+url.PathEscape(instance.Id)+"/maintenance-procedures/update/parameters")
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
		err = json.NewDecoder(resp.Body).Decode(&parameters)
		if err != nil {
			t.Error(err)
			return
		}
		expectedParameters := []model.SmartServiceExtendedParameter{
			{
				SmartServiceParameter: model.SmartServiceParameter{
					Id:    "pipeline_id",
					Label: "pipeline_id",
				},
				Type: "https://schema.org/Text",
			},
		}
		if !reflect.DeepEqual(parameters, expectedParameters) {
			actual, _ := json.Marshal(parameters)
			expected, _ := json.Marshal(expectedParameters)
			t.Error("\n", string(expected), "\n", string(actual))
		}
	})

	t.Run("start maintenance", func(t *testing.T) {
		resp, err := post(userToken, apiUrl+"/instances/"+url.PathEscape(instance.Id)+"/maintenance-procedures/update/start", []model.SmartServiceParameter{
			{
				Id:         "pipeline_id",
				Value:      "foobar",
				Label:      "pipeline_id",
				ValueLabel: "foobar",
			},
		})
		if err != nil {
			t.Error(err)
			return
		}
		if resp.StatusCode != http.StatusNoContent {
			temp, _ := io.ReadAll(resp.Body)
			t.Error(resp.StatusCode, string(temp))
			return
		}
	})

	time.Sleep(5 * time.Second)

	t.Run("read instance after maintenance", func(t *testing.T) {
		resp, err := get(userToken, apiUrl+"/instances/"+url.PathEscape(instance.Id))
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

		if instance.Name != "instance name" {
			t.Error(instance.Name)
			return
		}
		if instance.Description != "instance description" {
			t.Error(instance.Description)
			return
		}
		if instance.UserId != userId {
			t.Error(instance.UserId)
			return
		}
		if instance.DesignId != design.Id {
			t.Error(instance.DesignId)
			return
		}
		if instance.ReleaseId != release.Id {
			t.Error(instance.ReleaseId)
			return
		}
		if !instance.Ready {
			t.Error(instance.Ready)
			return
		}
		if instance.Error != "" {
			t.Error(instance.Error)
			return
		}
		if len(instance.RunningMaintenanceIds) > 0 {
			t.Error(instance.RunningMaintenanceIds)
			return
		}
	})
	t.Log(instance)

	t.Run("list modules", func(t *testing.T) {
		testModuleListExtended(t, apiUrl, "", 2, []string{instance.Id}, []string{"test-module", mocks.CAMUNDA_MODULE_WORKER_TOPIC}, func(modules []model.SmartServiceModule) {
			b, _ := json.Marshal(modules)
			t.Log(string(b))
		})
	})
}
