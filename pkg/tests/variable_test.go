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
	"github.com/SENERGY-Platform/smart-service-repository/pkg/camunda"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
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

func TestVariableApi(t *testing.T) {
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

	instanceA := model.SmartServiceInstance{}
	largeString := ""
	for i := 0; i < 5000; i++ {
		largeString = largeString + "a"
	}

	t.Run("create instance a with large parameter", func(t *testing.T) {
		resp, err := post(userToken, apiUrl+"/releases/"+url.PathEscape(release.Id)+"/instances", model.SmartServiceInstanceInit{
			SmartServiceInstanceInfo: model.SmartServiceInstanceInfo{
				Name:        "instance name",
				Description: "instance description",
			},
			Parameters: append(fillTestParameter(parameters), model.SmartServiceParameter{
				Id:         "large",
				Value:      largeString,
				Label:      "large",
				ValueLabel: "large",
			}),
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

		if instanceA.Name != "instance name" {
			t.Error(instanceA.Name)
			return
		}
		if instanceA.Error != "" {
			t.Error(instanceA.Error)
			return
		}
	})

	instanceB := model.SmartServiceInstance{}
	t.Run("create instance b", func(t *testing.T) {
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

		if instanceB.Name != "instance name" {
			t.Error(instanceB.Name)
			return
		}
		if instanceB.Error != "" {
			t.Error(instanceB.Error)
			return
		}
	})

	processIdOfInstanceA := ""
	t.Run("read process instances", func(t *testing.T) {
		instances, err := camunda.New(config).GetProcessInstanceList()
		if err != nil {
			t.Error(err)
			return
		}
		for _, instance := range instances {
			if instance.BusinessKey == instanceA.Id {
				processIdOfInstanceA = instance.Id
			}
		}
	})

	t.Run("add variable to instance a", func(t *testing.T) {
		resp, err := put(userToken, apiUrl+"/instances/"+url.PathEscape(instanceA.Id)+"/variables/var_a_1_name", 42)
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

	t.Run("add variable map to process a", func(t *testing.T) {
		resp, err := put(adminToken, apiUrl+"/instances-by-process-id/"+url.PathEscape(processIdOfInstanceA)+"/variables-map", map[string]interface{}{
			"var_a_2_name": "str",
			"var_a_3_name": 13,
			"var_a_4_name": true,
			"var_a_5_name": "tobedeleted",
			"var_a_6_name": 2.4,
			"var_a_7_name": map[string]interface{}{"foo": "bar", "batz": 42},
			"var_a_8_name": nil,
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
	})

	t.Run("delete variable from instance a", func(t *testing.T) {
		resp, err := delete(userToken, apiUrl+"/instances/"+url.PathEscape(instanceA.Id)+"/variables/var_a_5_name")
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

	t.Run("read variable of instance a", func(t *testing.T) {
		expected := map[string]interface{}{
			"instance_id": instanceA.Id,
			"name":        "var_a_2_name",
			"user_id":     instanceA.UserId,
			"value":       "str",
		}
		resp, err := get(userToken, apiUrl+"/instances/"+url.PathEscape(instanceA.Id)+"/variables/var_a_2_name")
		if err != nil {
			t.Error(err)
			return
		}
		if resp.StatusCode != http.StatusOK {
			temp, _ := io.ReadAll(resp.Body)
			t.Error(resp.StatusCode, string(temp))
			return
		}
		var actual interface{}
		err = json.NewDecoder(resp.Body).Decode(&actual)
		if err != nil {
			t.Error(err)
			return
		}
		if !reflect.DeepEqual(expected, actual) {
			t.Errorf("\nexpected: %#v\nactual:%#v\n", expected, actual)
		}
	})

	t.Run("list instance a variables", func(t *testing.T) {
		expected := []interface{}{
			map[string]interface{}{
				"instance_id": instanceA.Id,
				"user_id":     instanceA.UserId,
				"name":        "color_hex",
				"value":       "#ff00ff",
			},
			map[string]interface{}{
				"instance_id": instanceA.Id,
				"user_id":     instanceA.UserId,
				"name":        "device_selection",
				"value":       "{\"device_selection\":{\"device_id\":\"device_1\",\"service_id\":\"s1\",\"path\":null},\"label\":\"Device 1: one service, no paths\"}",
			},
			map[string]interface{}{
				"instance_id": instanceA.Id,
				"user_id":     instanceA.UserId,
				"name":        "large",
				"value":       largeString,
			},
			map[string]interface{}{
				"instance_id": instanceA.Id,
				"user_id":     instanceA.UserId,
				"name":        "var_a_1_name",
				"value":       float64(42),
			},
			map[string]interface{}{
				"instance_id": instanceA.Id,
				"user_id":     instanceA.UserId,
				"name":        "var_a_2_name",
				"value":       "str",
			},
			map[string]interface{}{
				"instance_id": instanceA.Id,
				"user_id":     instanceA.UserId,
				"name":        "var_a_3_name",
				"value":       float64(13),
			},
			map[string]interface{}{
				"instance_id": instanceA.Id,
				"user_id":     instanceA.UserId,
				"name":        "var_a_4_name",
				"value":       true,
			},
			map[string]interface{}{
				"instance_id": instanceA.Id,
				"user_id":     instanceA.UserId,
				"name":        "var_a_6_name",
				"value":       2.4,
			},
			map[string]interface{}{
				"instance_id": instanceA.Id,
				"user_id":     instanceA.UserId,
				"name":        "var_a_7_name",
				"value":       map[string]interface{}{"foo": "bar", "batz": float64(42)},
			},
			map[string]interface{}{
				"instance_id": instanceA.Id,
				"user_id":     instanceA.UserId,
				"name":        "var_a_8_name",
				"value":       nil,
			},
		}
		resp, err := get(userToken, apiUrl+"/instances/"+url.PathEscape(instanceA.Id)+"/variables")
		if err != nil {
			t.Error(err)
			return
		}
		if resp.StatusCode != http.StatusOK {
			temp, _ := io.ReadAll(resp.Body)
			t.Error(resp.StatusCode, string(temp))
			return
		}
		var actual interface{}
		err = json.NewDecoder(resp.Body).Decode(&actual)
		if err != nil {
			t.Error(err)
			return
		}
		if !reflect.DeepEqual(expected, actual) {
			t.Errorf("\nexpected:%#v\nactual:%#v\n", expected, actual)
		}
	})

	t.Run("list instance a variables as map", func(t *testing.T) {
		expected := map[string]interface{}{
			"color_hex":        "#ff00ff",
			"device_selection": "{\"device_selection\":{\"device_id\":\"device_1\",\"service_id\":\"s1\",\"path\":null},\"label\":\"Device 1: one service, no paths\"}",
			"large":            largeString,
			"var_a_1_name":     float64(42),
			"var_a_2_name":     "str",
			"var_a_3_name":     float64(13),
			"var_a_4_name":     true,
			"var_a_6_name":     2.4,
			"var_a_7_name":     map[string]interface{}{"foo": "bar", "batz": float64(42)},
			"var_a_8_name":     nil,
		}
		resp, err := get(userToken, apiUrl+"/instances/"+url.PathEscape(instanceA.Id)+"/variables-map")
		if err != nil {
			t.Error(err)
			return
		}
		if resp.StatusCode != http.StatusOK {
			temp, _ := io.ReadAll(resp.Body)
			t.Error(resp.StatusCode, string(temp))
			return
		}
		var actual interface{}
		err = json.NewDecoder(resp.Body).Decode(&actual)
		if err != nil {
			t.Error(err)
			return
		}
		if !reflect.DeepEqual(expected, actual) {
			t.Errorf("\nexpected: %#v\nactual:%#v\n", expected, actual)
		}
	})

	t.Run("list process a variables as map", func(t *testing.T) {
		expected := map[string]interface{}{
			"color_hex":        "#ff00ff",
			"device_selection": "{\"device_selection\":{\"device_id\":\"device_1\",\"service_id\":\"s1\",\"path\":null},\"label\":\"Device 1: one service, no paths\"}",
			"large":            largeString,
			"var_a_1_name":     float64(42),
			"var_a_2_name":     "str",
			"var_a_3_name":     float64(13),
			"var_a_4_name":     true,
			"var_a_6_name":     2.4,
			"var_a_7_name":     map[string]interface{}{"foo": "bar", "batz": float64(42)},
			"var_a_8_name":     nil,
		}
		resp, err := get(adminToken, apiUrl+"/instances-by-process-id/"+url.PathEscape(processIdOfInstanceA)+"/variables-map")
		if err != nil {
			t.Error(err)
			return
		}
		if resp.StatusCode != http.StatusOK {
			temp, _ := io.ReadAll(resp.Body)
			t.Error(resp.StatusCode, string(temp))
			return
		}
		var actual interface{}
		err = json.NewDecoder(resp.Body).Decode(&actual)
		if err != nil {
			t.Error(err)
			return
		}
		if !reflect.DeepEqual(expected, actual) {
			t.Errorf("\nexpected: %#v\nactual:%#v\n", expected, actual)
		}
	})

	t.Run("list instance b variables", func(t *testing.T) {

	})

	t.Run("delete instance b", func(t *testing.T) {
		resp, err := delete(userToken, apiUrl+"/instances/"+url.PathEscape(instanceB.Id))
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

	t.Run("list instance b variables after delete", func(t *testing.T) {
		expected := []interface{}{}
		resp, err := get(userToken, apiUrl+"/instances/"+url.PathEscape(instanceB.Id)+"/variables")
		if err != nil {
			t.Error(err)
			return
		}
		if resp.StatusCode != http.StatusOK {
			temp, _ := io.ReadAll(resp.Body)
			t.Error(resp.StatusCode, string(temp))
			return
		}
		var actual interface{}
		err = json.NewDecoder(resp.Body).Decode(&actual)
		if err != nil {
			t.Error(err)
			return
		}
		if !reflect.DeepEqual(expected, actual) {
			t.Errorf("\nexpected: %#v\nactual:%#v\n", expected, actual)
		}
	})

	t.Run("list instance a variables after delete", func(t *testing.T) {
		expected := []interface{}{
			map[string]interface{}{
				"instance_id": instanceA.Id,
				"user_id":     instanceA.UserId,
				"name":        "color_hex",
				"value":       "#ff00ff",
			},
			map[string]interface{}{
				"instance_id": instanceA.Id,
				"user_id":     instanceA.UserId,
				"name":        "device_selection",
				"value":       "{\"device_selection\":{\"device_id\":\"device_1\",\"service_id\":\"s1\",\"path\":null},\"label\":\"Device 1: one service, no paths\"}",
			},
			map[string]interface{}{
				"instance_id": instanceA.Id,
				"user_id":     instanceA.UserId,
				"name":        "large",
				"value":       largeString,
			},
			map[string]interface{}{
				"instance_id": instanceA.Id,
				"user_id":     instanceA.UserId,
				"name":        "var_a_1_name",
				"value":       float64(42),
			},
			map[string]interface{}{
				"instance_id": instanceA.Id,
				"user_id":     instanceA.UserId,
				"name":        "var_a_2_name",
				"value":       "str",
			},
			map[string]interface{}{
				"instance_id": instanceA.Id,
				"user_id":     instanceA.UserId,
				"name":        "var_a_3_name",
				"value":       float64(13),
			},
			map[string]interface{}{
				"instance_id": instanceA.Id,
				"user_id":     instanceA.UserId,
				"name":        "var_a_4_name",
				"value":       true,
			},
			map[string]interface{}{
				"instance_id": instanceA.Id,
				"user_id":     instanceA.UserId,
				"name":        "var_a_6_name",
				"value":       2.4,
			},
			map[string]interface{}{
				"instance_id": instanceA.Id,
				"user_id":     instanceA.UserId,
				"name":        "var_a_7_name",
				"value":       map[string]interface{}{"foo": "bar", "batz": float64(42)},
			},
			map[string]interface{}{
				"instance_id": instanceA.Id,
				"user_id":     instanceA.UserId,
				"name":        "var_a_8_name",
				"value":       nil,
			},
		}
		resp, err := get(userToken, apiUrl+"/instances/"+url.PathEscape(instanceA.Id)+"/variables")
		if err != nil {
			t.Error(err)
			return
		}
		if resp.StatusCode != http.StatusOK {
			temp, _ := io.ReadAll(resp.Body)
			t.Error(resp.StatusCode, string(temp))
			return
		}
		var actual interface{}
		err = json.NewDecoder(resp.Body).Decode(&actual)
		if err != nil {
			t.Error(err)
			return
		}
		if !reflect.DeepEqual(expected, actual) {
			t.Errorf("\nexpected: %#v\nactual:%#v\n", expected, actual)
		}
	})
}

// SNRGY-2756: iot options may not be nil. they have to be at least an empty list.
func TestNoDeviceOptionVariableApi(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	apiUrl, _, err := apiTestEnv(ctx, wg, true, []model.Selectable{}, func(err error) {
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
			BpmnXml: resources.ProcessDeploymentNoDeviceOptionBpmn,
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
		if design.BpmnXml != resources.ProcessDeploymentNoDeviceOptionBpmn {
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
		parameters := []model.SmartServiceExtendedParameter{}
		err = json.NewDecoder(resp.Body).Decode(&parameters)
		if err != nil {
			t.Error(err)
			return
		}

		if !reflect.DeepEqual(resources.ExpectedNoDeviceOptionParamsObj, parameters) {
			temp1, _ := json.Marshal(parameters)
			temp2, _ := json.Marshal(resources.ExpectedNoDeviceOptionParamsObj)
			t.Error("\n", string(temp1), "\n", string(temp2))
		}

	})

}
