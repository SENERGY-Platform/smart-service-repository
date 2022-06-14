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
	"bytes"
	"context"
	"encoding/json"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/controller"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/tests/mocks"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/tests/resources"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestModuleApi(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	apiUrl, config, err := apiTestEnv(ctx, wg, true, nil, func(err error) {
		t.Error(err)
	})
	if err != nil {
		t.Error(err)
		return
	}

	design := model.SmartServiceDesign{}
	t.Run("create design", func(t *testing.T) {
		resp, err := post(userToken, apiUrl+"/designs", model.SmartServiceDesign{
			BpmnXml: resources.ProcessDeploymentBpmn,
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
		if design.BpmnXml != resources.ProcessDeploymentBpmn {
			t.Error(design.BpmnXml)
			return
		}
		if design.SvgXml != resources.ProcessDeploymentSvg {
			t.Error(design.SvgXml)
			return
		}
		if design.Id == "" {
			t.Error(design.Id)
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

	parameters := []model.SmartServiceExtendedParameter{}
	t.Run("read params", func(t *testing.T) {
		resp, err := get(userToken, apiUrl+"/releases/"+url.PathEscape(release.Id)+"/parameters")
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
	})

	instanceA := model.SmartServiceInstance{}
	t.Run("create instance", func(t *testing.T) {
		resp, err := post(userToken, apiUrl+"/releases/"+url.PathEscape(release.Id)+"/instances", model.SmartServiceInstanceInit{
			SmartServiceInstanceInfo: model.SmartServiceInstanceInfo{
				Name:        "instance name",
				Description: "instance description",
			},
			Parameters: fillTestParameter(parameters),
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
		err = json.NewDecoder(resp.Body).Decode(&instanceA)
		if err != nil {
			t.Error(err)
			return
		}
	})

	instanceB := model.SmartServiceInstance{}
	t.Run("create instance", func(t *testing.T) {
		resp, err := post(userToken, apiUrl+"/releases/"+url.PathEscape(release.Id)+"/instances", model.SmartServiceInstanceInit{
			SmartServiceInstanceInfo: model.SmartServiceInstanceInfo{
				Name:        "instance name",
				Description: "instance description",
			},
			Parameters: fillTestParameter(parameters),
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
		err = json.NewDecoder(resp.Body).Decode(&instanceB)
		if err != nil {
			t.Error(err)
			return
		}
	})

	//time.Sleep(2 * time.Second)

	mocks.NewModuleWorker(ctx, wg, apiUrl, config, func(taskWorkerMsg mocks.ModuleWorkerMessage) (err error) {
		expectedVariables := map[string]mocks.CamundaVariable{
			"Task_foo.parameter": {
				Type:  "String",
				Value: "{\"inputs.on\": true, \"inputs.hex\": #ff00ff}",
			},
			"Task_foo.selection": {
				Type:  "String",
				Value: "{\"device_selection\":{\"device_id\":\"device_1\",\"service_id\":\"s1\",\"path\":null}}",
			},
			"color_hex": {
				Type:  "String",
				Value: "#ff00ff",
			},
			"device_selection": {
				Type:  "String",
				Value: "{\"device_selection\":{\"device_id\":\"device_1\",\"service_id\":\"s1\",\"path\":null}}",
			},
			"process_model_id": {
				Type:  "String",
				Value: "76e6f65c-c3c1-47c0-a999-4675baace425",
			},
		}
		temp, _ := json.Marshal(taskWorkerMsg.Variables)
		t.Log("worker call:", string(temp))
		if !reflect.DeepEqual(taskWorkerMsg.Variables, expectedVariables) {
			t.Error(string(temp))
		}
		return nil
	})

	time.Sleep(2 * time.Second)

	receivedModuleDelete := false
	mockDelete := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		receivedModuleDelete = true
		writer.WriteHeader(200)
	}))
	wg.Add(1)
	go func() {
		<-ctx.Done()
		mockDelete.Close()
		wg.Done()
	}()

	t.Run("set instance module", func(t *testing.T) {
		//other modules are set by the worker mock
		module := model.SmartServiceModuleInit{
			ModuleType: "test-module",
			ModuleData: map[string]interface{}{
				"foo": "bar",
			},
			DeleteInfo: &model.ModuleDeleteInfo{
				Url:    mockDelete.URL,
				UserId: userId,
			},
		}

		body := new(bytes.Buffer)
		err := json.NewEncoder(body).Encode(module)
		if err != nil {
			t.Error(err)
			return
		}
		req, err := http.NewRequest("POST", apiUrl+"/instances/"+url.PathEscape(instanceA.Id)+"/modules", body)
		if err != nil {
			t.Error(err)
			return
		}
		req.Header.Set("Authorization", userToken)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
			return
		}
		if resp.StatusCode >= 300 {
			temp, _ := io.ReadAll(resp.Body)
			t.Error(resp.StatusCode, string(temp))
			return
		}
	})

	t.Run("list all modules", func(t *testing.T) {
		testModuleList(t, apiUrl, "", 3, []string{instanceA.Id, instanceB.Id}, []string{"test-module", mocks.CAMUNDA_MODULE_WORKER_TOPIC})
	})

	t.Run("list instance a modules", func(t *testing.T) {
		testModuleList(t, apiUrl, "?instance_id="+url.QueryEscape(instanceA.Id), 2, []string{instanceA.Id}, []string{"test-module", mocks.CAMUNDA_MODULE_WORKER_TOPIC})
	})

	t.Run("list instance b modules", func(t *testing.T) {
		testModuleList(t, apiUrl, "?instance_id="+url.QueryEscape(instanceB.Id), 1, []string{instanceB.Id}, []string{"test-module", mocks.CAMUNDA_MODULE_WORKER_TOPIC})
	})

	t.Run("list test-type modules", func(t *testing.T) {
		testModuleList(t, apiUrl, "?module_type="+url.QueryEscape("test-module"), 1, []string{instanceA.Id, instanceB.Id}, []string{"test-module"})
	})

	t.Run("list mock-type modules", func(t *testing.T) {
		testModuleList(t, apiUrl, "?module_type="+url.QueryEscape(mocks.CAMUNDA_MODULE_WORKER_TOPIC), 2, []string{instanceA.Id, instanceB.Id}, []string{mocks.CAMUNDA_MODULE_WORKER_TOPIC})
	})

	t.Run("delete instance", func(t *testing.T) {
		resp, err := delete(userToken, apiUrl+"/instances/"+url.PathEscape(instanceA.Id))
		if err != nil {
			t.Error(err)
			return
		}
		if resp.StatusCode != http.StatusOK {
			temp, _ := io.ReadAll(resp.Body)
			t.Error(resp.StatusCode, string(temp))
			return
		}
	})

	t.Run("check module delete on instance delete", func(t *testing.T) {
		if !receivedModuleDelete {
			t.Error(receivedModuleDelete)
		}
	})
}

func TestModuleBulkApi(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	apiUrl, config, err := apiTestEnv(ctx, wg, true, nil, func(err error) {
		t.Error(err)
	})
	if err != nil {
		t.Error(err)
		return
	}

	design := model.SmartServiceDesign{}
	t.Run("create design", func(t *testing.T) {
		resp, err := post(userToken, apiUrl+"/designs", model.SmartServiceDesign{
			BpmnXml: resources.ProcessDeploymentBpmn,
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
		if design.BpmnXml != resources.ProcessDeploymentBpmn {
			t.Error(design.BpmnXml)
			return
		}
		if design.SvgXml != resources.ProcessDeploymentSvg {
			t.Error(design.SvgXml)
			return
		}
		if design.Id == "" {
			t.Error(design.Id)
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

	parameters := []model.SmartServiceExtendedParameter{}
	t.Run("read params", func(t *testing.T) {
		resp, err := get(userToken, apiUrl+"/releases/"+url.PathEscape(release.Id)+"/parameters")
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
	})

	instance := model.SmartServiceInstance{}
	t.Run("create instance", func(t *testing.T) {
		resp, err := post(userToken, apiUrl+"/releases/"+url.PathEscape(release.Id)+"/instances", model.SmartServiceInstanceInit{
			SmartServiceInstanceInfo: model.SmartServiceInstanceInfo{
				Name:        "instance name",
				Description: "instance description",
			},
			Parameters: fillTestParameter(parameters),
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

	processInstanceId := ""
	mocks.NewModuleWorker(ctx, wg, apiUrl, config, func(taskWorkerMsg mocks.ModuleWorkerMessage) (err error) {
		processInstanceId = taskWorkerMsg.ProcessInstanceId
		expectedVariables := map[string]mocks.CamundaVariable{
			"Task_foo.parameter": {
				Type:  "String",
				Value: "{\"inputs.on\": true, \"inputs.hex\": #ff00ff}",
			},
			"Task_foo.selection": {
				Type:  "String",
				Value: "{\"device_selection\":{\"device_id\":\"device_1\",\"service_id\":\"s1\",\"path\":null}}",
			},
			"color_hex": {
				Type:  "String",
				Value: "#ff00ff",
			},
			"device_selection": {
				Type:  "String",
				Value: "{\"device_selection\":{\"device_id\":\"device_1\",\"service_id\":\"s1\",\"path\":null}}",
			},
			"process_model_id": {
				Type:  "String",
				Value: "76e6f65c-c3c1-47c0-a999-4675baace425",
			},
		}
		temp, _ := json.Marshal(taskWorkerMsg.Variables)
		t.Log("worker call:", string(temp))
		if !reflect.DeepEqual(taskWorkerMsg.Variables, expectedVariables) {
			t.Error(string(temp))
		}
		return nil
	})

	time.Sleep(2 * time.Second)

	t.Run("set instance modules", func(t *testing.T) {
		//other modules are set by the worker mock
		modules := []model.SmartServiceModuleInit{
			{
				ModuleType: "test-module-1",
				ModuleData: map[string]interface{}{
					"foo": "bar",
				},
			},
			{
				ModuleType: "test-module-2",
				ModuleData: map[string]interface{}{
					"foo": "bar",
				},
			},
		}

		body := new(bytes.Buffer)
		err := json.NewEncoder(body).Encode(modules)
		if err != nil {
			t.Error(err)
			return
		}
		req, err := http.NewRequest("POST", apiUrl+"/instances-by-process-id/"+url.PathEscape(processInstanceId)+"/modules/bulk", body)
		if err != nil {
			t.Error(err)
			return
		}
		req.Header.Set("Authorization", adminToken)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
			return
		}
		if resp.StatusCode >= 300 {
			temp, _ := io.ReadAll(resp.Body)
			t.Error(resp.StatusCode, string(temp))
			return
		}
	})

	t.Run("list all modules", func(t *testing.T) {
		testModuleList(t, apiUrl, "", 3, []string{instance.Id}, []string{"test-module-1", "test-module-2", mocks.CAMUNDA_MODULE_WORKER_TOPIC})
	})

	t.Run("list instance modules", func(t *testing.T) {
		testModuleList(t, apiUrl, "?instance_id="+url.QueryEscape(instance.Id), 3, []string{instance.Id}, []string{"test-module-1", "test-module-2", mocks.CAMUNDA_MODULE_WORKER_TOPIC})
	})

	t.Run("list test-type 1 modules", func(t *testing.T) {
		testModuleList(t, apiUrl, "?module_type="+url.QueryEscape("test-module-1"), 1, []string{instance.Id}, []string{"test-module-1"})
	})

	t.Run("list test-type 2 modules", func(t *testing.T) {
		testModuleList(t, apiUrl, "?module_type="+url.QueryEscape("test-module-2"), 1, []string{instance.Id}, []string{"test-module-2"})
	})

}

func testModuleList(t *testing.T, apiUrl string, query string, expectedCount int, allowedInstanceIds []string, allowedModuleTypes []string) {
	t.Helper()
	resp, err := get(userToken, apiUrl+"/modules"+query)
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
	result := []model.SmartServiceModule{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		t.Error(err)
		return
	}
	if len(result) != expectedCount {
		t.Error(len(result), expectedCount, result)
		return
	}
	for _, element := range result {
		if !controller.ListContains(allowedInstanceIds, func(s string) bool { return s == element.InstanceId }) {
			t.Error(element.InstanceId, allowedInstanceIds)
			return
		}
		if !controller.ListContains(allowedModuleTypes, func(s string) bool { return s == element.ModuleType }) {
			t.Error(element.InstanceId, allowedInstanceIds)
			return
		}
		t.Log(element.ModuleData)
	}
}
