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
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/tests/mocks"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/tests/resources"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"runtime/debug"
	"slices"
	"sync"
	"testing"
	"time"
)

func TestModuleApi(t *testing.T) {
	if CI {
		t.Skip("not in ci")
	}
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	apiUrl, config, _, err := apiTestEnv(ctx, wg, true, nil, func(err error) {
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
				Value: "{\"device_selection\":{\"device_id\":\"device_1\",\"service_id\":\"s1\",\"path\":null},\"label\":\"Device 1: one service, no paths\"}",
			},
			"color_hex": {
				Type:  "String",
				Value: "#ff00ff",
			},
			"device_selection": {
				Type:  "String",
				Value: "{\"device_selection\":{\"device_id\":\"device_1\",\"service_id\":\"s1\",\"path\":null},\"label\":\"Device 1: one service, no paths\"}",
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

func TestModuleKeyApi(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	apiUrl, config, _, err := apiTestEnv(ctx, wg, true, nil, func(err error) {
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

	processInstanceIds := []string{}
	mocks.NewModuleWorker(ctx, wg, apiUrl, config, func(taskWorkerMsg mocks.ModuleWorkerMessage) (err error) {
		processInstanceIds = append(processInstanceIds, taskWorkerMsg.ProcessInstanceId)
		expectedVariables := map[string]mocks.CamundaVariable{
			"Task_foo.parameter": {
				Type:  "String",
				Value: "{\"inputs.on\": true, \"inputs.hex\": #ff00ff}",
			},
			"Task_foo.selection": {
				Type:  "String",
				Value: "{\"device_selection\":{\"device_id\":\"device_1\",\"service_id\":\"s1\",\"path\":null},\"label\":\"Device 1: one service, no paths\"}",
			},
			"color_hex": {
				Type:  "String",
				Value: "#ff00ff",
			},
			"device_selection": {
				Type:  "String",
				Value: "{\"device_selection\":{\"device_id\":\"device_1\",\"service_id\":\"s1\",\"path\":null},\"label\":\"Device 1: one service, no paths\"}",
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
			Keys: []string{"modulekey", "foo=bar"},
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

	t.Run("list by key="+url.QueryEscape("foo=bar"), func(t *testing.T) {
		testModuleListExtended(t, apiUrl, "?key="+url.QueryEscape("foo=bar"), 1, []string{instanceA.Id}, []string{"test-module"}, func(modules []model.SmartServiceModule) {
			if len(modules) != 1 || modules[0].InstanceId != instanceA.Id || len(modules[0].Keys) != 2 || modules[0].Keys[1] != "foo=bar" {
				b, _ := json.Marshal(modules)
				t.Error(instanceA.Id, string(b))
			}
		})
	})

	t.Run("list by key=modulekey", func(t *testing.T) {
		testModuleListExtended(t, apiUrl, "?key=modulekey", 1, []string{instanceA.Id}, []string{"test-module"}, func(modules []model.SmartServiceModule) {
			if len(modules) != 1 || modules[0].InstanceId != instanceA.Id || len(modules[0].Keys) != 2 || modules[0].Keys[0] != "modulekey" {
				b, _ := json.Marshal(modules)
				t.Error(instanceA.Id, string(b))
			}
		})
	})

	processInstanceId := ""
	t.Run("find process-instance id of instanceA", func(t *testing.T) {
		for _, id := range processInstanceIds {
			req, err := http.NewRequest("GET", apiUrl+"/instances-by-process-id/"+url.PathEscape(id), nil)
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
			respInstance := model.SmartServiceInstance{}
			err = json.NewDecoder(resp.Body).Decode(&respInstance)
			if err != nil {
				t.Error(err)
				return
			}
			if respInstance.Id == instanceA.Id {
				processInstanceId = id
				return
			}
		}
		t.Error("no matching process instance id found")
	})

	t.Run("list by process-instance-id and key="+url.QueryEscape("foo=bar"), func(t *testing.T) {
		testModuleListExtendedWithToken(t, adminToken, apiUrl+"/instances-by-process-id/"+url.PathEscape(processInstanceId), "?key="+url.QueryEscape("foo=bar"), 1, []string{instanceA.Id}, []string{"test-module"}, func(modules []model.SmartServiceModule) {
			if len(modules) != 1 || modules[0].InstanceId != instanceA.Id || len(modules[0].Keys) != 2 || modules[0].Keys[1] != "foo=bar" {
				b, _ := json.Marshal(modules)
				t.Error(instanceA.Id, string(b))
			}
		})
	})

	t.Run("list by process-instance-id and key=modulekey", func(t *testing.T) {
		testModuleListExtendedWithToken(t, adminToken, apiUrl+"/instances-by-process-id/"+url.PathEscape(processInstanceId), "?key=modulekey", 1, []string{instanceA.Id}, []string{"test-module"}, func(modules []model.SmartServiceModule) {
			if len(modules) != 1 || modules[0].InstanceId != instanceA.Id || len(modules[0].Keys) != 2 || modules[0].Keys[0] != "modulekey" {
				b, _ := json.Marshal(modules)
				t.Error(instanceA.Id, string(b))
			}
		})
	})

	t.Run("list by process-instance-id and key=modulekey and module_type=test-module", func(t *testing.T) {
		testModuleListExtendedWithToken(t, adminToken, apiUrl+"/instances-by-process-id/"+url.PathEscape(processInstanceId), "?key=modulekey&module_type=test-module", 1, []string{instanceA.Id}, []string{"test-module"}, func(modules []model.SmartServiceModule) {
			if len(modules) != 1 || modules[0].InstanceId != instanceA.Id || len(modules[0].Keys) != 2 || modules[0].Keys[0] != "modulekey" {
				b, _ := json.Marshal(modules)
				t.Error(instanceA.Id, string(b))
			}
		})
	})

	t.Run("list by process-instance-id", func(t *testing.T) {
		testModuleListExtendedWithToken(t, adminToken, apiUrl+"/instances-by-process-id/"+url.PathEscape(processInstanceId), "", 2, []string{instanceA.Id}, []string{"test-module", mocks.CAMUNDA_MODULE_WORKER_TOPIC}, func(modules []model.SmartServiceModule) {

		})
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

func TestModulePutApi(t *testing.T) {
	if CI {
		t.Skip("not in ci")
	}
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	apiUrl, config, _, err := apiTestEnv(ctx, wg, true, nil, func(err error) {
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

	//time.Sleep(2 * time.Second)

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
				Value: "{\"device_selection\":{\"device_id\":\"device_1\",\"service_id\":\"s1\",\"path\":null},\"label\":\"Device 1: one service, no paths\"}",
			},
			"color_hex": {
				Type:  "String",
				Value: "#ff00ff",
			},
			"device_selection": {
				Type:  "String",
				Value: "{\"device_selection\":{\"device_id\":\"device_1\",\"service_id\":\"s1\",\"path\":null},\"label\":\"Device 1: one service, no paths\"}",
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

	t.Run("check instance user", func(t *testing.T) {
		req, err := http.NewRequest("GET", apiUrl+"/instances-by-process-id/"+url.PathEscape(processInstanceId)+"/user-id", nil)
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
		respUserId := ""
		err = json.NewDecoder(resp.Body).Decode(&respUserId)
		if err != nil {
			t.Error(err)
			return
		}
		if respUserId != userId {
			t.Error(respUserId, userId)
			return
		}
	})

	t.Run("check instance by process id", func(t *testing.T) {
		req, err := http.NewRequest("GET", apiUrl+"/instances-by-process-id/"+url.PathEscape(processInstanceId), nil)
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
		respInstance := model.SmartServiceInstance{}
		err = json.NewDecoder(resp.Body).Decode(&respInstance)
		if err != nil {
			t.Error(err)
			return
		}
		if respInstance.UserId != userId {
			t.Error(respInstance.UserId, userId)
			return
		}
		if respInstance.Id != instanceA.Id {
			t.Error(respInstance.Id, instanceA.Id)
			return
		}
	})

	t.Run("create instance module", func(t *testing.T) {
		//other modules are set by the worker mock
		module := model.SmartServiceModuleInit{
			ModuleType: "test-module-old",
			ModuleData: map[string]interface{}{
				"foo": "bar",
			},
		}

		body := new(bytes.Buffer)
		err := json.NewEncoder(body).Encode(module)
		if err != nil {
			t.Error(err)
			return
		}
		req, err := http.NewRequest("PUT", apiUrl+"/instances-by-process-id/"+url.PathEscape(processInstanceId)+"/modules/foo-bar", body)
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

	t.Run("overwrite instance module overwrite", func(t *testing.T) {
		//other modules are set by the worker mock
		module := model.SmartServiceModuleInit{
			ModuleType: "test-module",
			ModuleData: map[string]interface{}{
				"foo": "bar",
			},
		}

		body := new(bytes.Buffer)
		err := json.NewEncoder(body).Encode(module)
		if err != nil {
			t.Error(err)
			return
		}
		req, err := http.NewRequest("PUT", apiUrl+"/instances-by-process-id/"+url.PathEscape(processInstanceId)+"/modules/foo-bar", body)
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
		testModuleList(t, apiUrl, "", 2, []string{instanceA.Id}, []string{"test-module", mocks.CAMUNDA_MODULE_WORKER_TOPIC})
	})

	t.Run("list instance a modules", func(t *testing.T) {
		testModuleList(t, apiUrl, "?instance_id="+url.QueryEscape(instanceA.Id), 2, []string{instanceA.Id}, []string{"test-module", mocks.CAMUNDA_MODULE_WORKER_TOPIC})
	})

	t.Run("list test-type modules", func(t *testing.T) {
		testModuleList(t, apiUrl, "?module_type="+url.QueryEscape("test-module"), 1, []string{instanceA.Id}, []string{"test-module"})
	})

	t.Run("list mock-type modules", func(t *testing.T) {
		testModuleList(t, apiUrl, "?module_type="+url.QueryEscape(mocks.CAMUNDA_MODULE_WORKER_TOPIC), 1, []string{instanceA.Id}, []string{mocks.CAMUNDA_MODULE_WORKER_TOPIC})
	})
}

func testModuleList(t *testing.T, apiUrl string, query string, expectedCount int, allowedInstanceIds []string, allowedModuleTypes []string) {
	testModuleListExtended(t, apiUrl, query, expectedCount, allowedInstanceIds, allowedModuleTypes, func(modules []model.SmartServiceModule) {})
}

func testModuleListExtended(t *testing.T, apiUrl string, query string, expectedCount int, allowedInstanceIds []string, allowedModuleTypes []string, checkResult func([]model.SmartServiceModule)) {
	testModuleListExtendedWithToken(t, userToken, apiUrl, query, expectedCount, allowedInstanceIds, allowedModuleTypes, checkResult)
	return
}

func testModuleListExtendedWithToken(t *testing.T, token string, apiUrl string, query string, expectedCount int, allowedInstanceIds []string, allowedModuleTypes []string, checkResult func([]model.SmartServiceModule)) {
	resp, err := get(token, apiUrl+"/modules"+query)
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
		if !slices.ContainsFunc(allowedInstanceIds, func(s string) bool { return s == element.InstanceId }) {
			t.Error(element.InstanceId, allowedInstanceIds)
			return
		}
		if !slices.ContainsFunc(allowedModuleTypes, func(s string) bool { return s == element.ModuleType }) {
			t.Error(element.InstanceId, allowedInstanceIds)
			return
		}
	}
	checkResult(result)
}

func TestEmptyAnalyticsVariables(t *testing.T) {
	if CI {
		t.Skip("not in ci")
	}
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	apiUrl, config, _, err := apiTestEnv(ctx, wg, true, nil, func(err error) {
		debug.PrintStack()
		t.Error(err)
	})
	if err != nil {
		t.Error(err)
		return
	}

	topicBackup := mocks.CAMUNDA_MODULE_WORKER_TOPIC
	defer func() {
		mocks.CAMUNDA_MODULE_WORKER_TOPIC = topicBackup
	}()
	mocks.CAMUNDA_MODULE_WORKER_TOPIC = "analytics"

	mocks.NewModuleWorker(ctx, wg, apiUrl, config, func(taskWorkerMsg mocks.ModuleWorkerMessage) (err error) {
		temp, _ := json.Marshal(taskWorkerMsg.Variables)
		t.Log("worker call:", string(temp))
		return nil
	})

	design := model.SmartServiceDesign{}
	t.Run("create design", func(t *testing.T) {
		resp, err := post(userToken, apiUrl+"/designs", model.SmartServiceDesign{
			BpmnXml: resources.EmptyAnalyticsTestBpmn,
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
			Parameters: []model.SmartServiceParameter{},
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
