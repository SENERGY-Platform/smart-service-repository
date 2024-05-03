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
	"errors"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/tests/mocks"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/tests/resources"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"runtime/debug"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestInstanceEditApi(t *testing.T) {
	if CI {
		t.Skip("not in ci")
	}
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	apiUrl, config, err := apiTestEnv(ctx, wg, true, nil, func(err error) {
		debug.PrintStack()
		t.Error(err)
	})
	if err != nil {
		t.Error(err)
		return
	}

	callCount := 0
	mocks.NewModuleWorker(ctx, wg, apiUrl, config, func(taskWorkerMsg mocks.ModuleWorkerMessage) (err error) {
		callCount = callCount + 1
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
		if callCount > 1 {
			expectedVariables["update"] = mocks.CamundaVariable{
				Type:  "String",
				Value: "foo",
			}
		}
		temp, _ := json.Marshal(taskWorkerMsg.Variables)
		t.Log("worker call:", string(temp))
		if !reflect.DeepEqual(taskWorkerMsg.Variables, expectedVariables) {
			t.Error(string(temp))
		}
		return nil
	})

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

	design2 := model.SmartServiceDesign{}
	t.Run("create updated design", func(t *testing.T) {
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
		err = json.NewDecoder(resp.Body).Decode(&design2)
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

	release2 := model.SmartServiceRelease{}
	t.Run("create updated release", func(t *testing.T) {
		resp, err := post(userToken, apiUrl+"/releases", model.SmartServiceRelease{
			DesignId:    design2.Id,
			Name:        "updated release",
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
		err = json.NewDecoder(resp.Body).Decode(&release2)
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

		if instance.Name != "instance name" {
			t.Error(instance.Name)
			return
		}
		if instance.Description != "instance description" {
			t.Error(instance.Description)
			return
		}
		if instance.UserId != userId {
			t.Error(instance.UserId, userId)
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
		if instance.Ready != false {
			t.Error(instance.Ready)
			return
		}
		if instance.Error != "" {
			t.Error(instance.Error)
			return
		}
	})

	t.Run("read instance", func(t *testing.T) {
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
		if instance.Ready != false {
			t.Error(instance.Ready)
			return
		}
	})

	t.Run("update instance name", func(t *testing.T) {
		resp, err := put(userToken, apiUrl+"/instances/"+url.PathEscape(instance.Id)+"/info", model.SmartServiceInstanceInfo{
			Name:        "instance name update",
			Description: "instance description update",
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

		if instance.Name != "instance name update" {
			t.Error(instance.Name)
			return
		}
		if instance.Description != "instance description update" {
			t.Error(instance.Description)
			return
		}
		if instance.UserId != userId {
			t.Error(instance.UserId, userId)
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
		if instance.Ready != false {
			t.Error(instance.Ready)
			return
		}
		if instance.Error != "" {
			t.Error(instance.Error)
			return
		}
	})

	t.Run("read instance update", func(t *testing.T) {
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

		if instance.Name != "instance name update" {
			t.Error(instance.Name)
			return
		}
		if instance.Description != "instance description update" {
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
		if instance.Ready != false {
			t.Error(instance.Ready)
			return
		}
	})

	time.Sleep(2 * time.Second)

	t.Run("update instance parameter", func(t *testing.T) {
		p := fillTestParameter(parameters)
		p = append(p, model.SmartServiceParameter{
			Id:    "update",
			Value: "foo",
		})
		resp, err := put(userToken, apiUrl+"/instances/"+url.PathEscape(instance.Id)+"/parameters", p)
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

		if instance.Name != "instance name update" {
			t.Error(instance.Name)
			return
		}
		if instance.Description != "instance description update" {
			t.Error(instance.Description)
			return
		}
		if instance.UserId != userId {
			t.Error(instance.UserId, userId)
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
		if instance.Ready != false {
			t.Error(instance.Ready)
			return
		}
		if instance.Error != "" {
			t.Error(instance.Error)
			return
		}
	})

	time.Sleep(2 * time.Second)

	t.Run("update instance release", func(t *testing.T) {
		p := fillTestParameter(parameters)
		p = append(p, model.SmartServiceParameter{
			Id:    "update",
			Value: "foo",
		})
		resp, err := put(userToken, apiUrl+"/instances/"+url.PathEscape(instance.Id)+"/parameters?release_id="+url.QueryEscape(release2.Id), p)
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

		if instance.Name != "instance name update" {
			t.Error(instance.Name)
			return
		}
		if instance.Description != "instance description update" {
			t.Error(instance.Description)
			return
		}
		if instance.UserId != userId {
			t.Error(instance.UserId, userId)
			return
		}
		if instance.DesignId != design2.Id {
			t.Error(instance.DesignId)
			return
		}
		if instance.ReleaseId != release2.Id {
			t.Error(instance.ReleaseId)
			return
		}
		if instance.Ready != false {
			t.Error(instance.Ready)
			return
		}
		if instance.Error != "" {
			t.Error(instance.Error)
			return
		}
	})

	t.Run("read instance after re design", func(t *testing.T) {
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

		if instance.Name != "instance name update" {
			t.Error(instance.Name)
			return
		}
		if instance.Description != "instance description update" {
			t.Error(instance.Description)
			return
		}
		if instance.UserId != userId {
			t.Error(instance.UserId)
			return
		}
		if instance.DesignId != design2.Id {
			t.Error(instance.DesignId)
			return
		}
		if instance.ReleaseId != release2.Id {
			t.Error(instance.ReleaseId)
			return
		}
		if instance.Ready != false {
			t.Error(instance.Ready)
			return
		}
	})

	time.Sleep(2 * time.Second)

	t.Run("check updated params", func(t *testing.T) {
		//real check of new parameter is in worker
		if callCount != 3 {
			t.Error("call count:", callCount)
		}
	})

	t.Run("delete release should error", func(t *testing.T) {
		resp, err := delete(userToken, apiUrl+"/releases/"+url.PathEscape(instance.ReleaseId))
		if err != nil {
			t.Error(err)
			return
		}
		if resp.StatusCode != http.StatusBadRequest {
			temp, _ := io.ReadAll(resp.Body)
			t.Error(resp.StatusCode, string(temp))
			return
		}
	})

	t.Run("delete instance", func(t *testing.T) {
		resp, err := delete(userToken, apiUrl+"/instances/"+url.PathEscape(instance.Id))
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

	t.Run("delete release should work", func(t *testing.T) {
		resp, err := delete(userToken, apiUrl+"/releases/"+url.PathEscape(instance.ReleaseId))
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
}

func TestInstanceApi(t *testing.T) {
	if CI {
		t.Skip("not in ci")
	}
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	apiUrl, config, err := apiTestEnv(ctx, wg, true, nil, func(err error) {
		debug.PrintStack()
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

		if instance.Name != "instance name" {
			t.Error(instance.Name)
			return
		}
		if instance.Description != "instance description" {
			t.Error(instance.Description)
			return
		}
		if instance.UserId != userId {
			t.Error(instance.UserId, userId)
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
		if instance.Ready != false {
			t.Error(instance.Ready)
			return
		}
		if instance.Error != "" {
			t.Error(instance.Error)
			return
		}
	})

	t.Run("read instance not ready", func(t *testing.T) {
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
		if instance.Ready != false {
			t.Error(instance.Ready)
			return
		}
	})

	count := 0
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
		if !reflect.DeepEqual(taskWorkerMsg.Variables, expectedVariables) {
			temp, _ := json.Marshal(taskWorkerMsg.Variables)
			t.Error(string(temp))
		}
		count = count + 1
		if count%2 == 0 {
			err = errors.New("test-error")
		}
		return err
	})

	names := []string{"a", "b", "c", "d", "e", "f"}
	t.Run("create list instances", func(t *testing.T) {
		for _, name := range names {
			resp, err := post(userToken, apiUrl+"/releases/"+url.PathEscape(release.Id)+"/instances", model.SmartServiceInstanceInit{
				SmartServiceInstanceInfo: model.SmartServiceInstanceInfo{
					Name:        name,
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
		}
	})

	time.Sleep(2 * time.Second) //allow mock worker to do its work

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

	t.Run("delete instance", func(t *testing.T) {
		resp, err := delete(userToken, apiUrl+"/instances/"+url.PathEscape(instance.Id))
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

	t.Run("list", func(t *testing.T) {
		t.Run("default", func(t *testing.T) {
			testInstanceList(t, apiUrl, "", names)
		})
		t.Run("sort=name.asc", func(t *testing.T) {
			testInstanceList(t, apiUrl, "?sort=name.asc", names)
		})
		t.Run("sort=name.desc", func(t *testing.T) {
			testInstanceList(t, apiUrl, "?sort=name.desc", reverse(names))
		})
		t.Run("limit and offset", func(t *testing.T) {
			testInstanceList(t, apiUrl, "?limit=2&offset=1", names[1:3])
		})
	})

	t.Run("read instance 404", func(t *testing.T) {
		resp, err := get(userToken, apiUrl+"/instances/foo")
		if err != nil {
			t.Error(err)
			return
		}
		if resp.StatusCode != http.StatusNotFound {
			temp, _ := io.ReadAll(resp.Body)
			t.Error(resp.StatusCode, string(temp))
			return
		}
		msg, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Error(err)
			return
		}
		if strings.Contains(string(msg), "mongo") {
			t.Error(string(msg))
			return
		}
	})

}

func fillTestParameter(parameters []model.SmartServiceExtendedParameter) (result []model.SmartServiceParameter) {
	for _, p := range parameters {
		p.Value = p.DefaultValue
		if p.Value == nil {
			if len(p.Options) > 0 {
				if p.Multiple {
					p.Value = []interface{}{p.Options[0].Value}
				} else {
					p.Value = p.Options[0].Value
				}
			}
		}
		result = append(result, p.SmartServiceParameter)
	}
	return result
}

func testInstanceList(t *testing.T, apiUrl string, query string, expectedNamesOrder []string) {
	resp, err := get(userToken, apiUrl+"/instances"+query)
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
	result := []model.SmartServiceInstance{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		t.Error(err)
		return
	}
	if len(result) != len(expectedNamesOrder) {
		t.Error(len(result), len(expectedNamesOrder), result)
		return
	}
	errCount := 0
	for i, element := range result {
		if element.Name != expectedNamesOrder[i] {
			t.Error(element.Name, expectedNamesOrder[i])
			return
		}
		if !element.Ready {
			t.Error(element.Name, element.Ready)
			return
		}
		if element.Error != "" {
			if element.Error != "test-error" {
				t.Error(element.Error)
			}
			errCount = errCount + 1
		}
	}
	if errCount != len(result)/2 {
		t.Error(errCount)
		return
	}
}
