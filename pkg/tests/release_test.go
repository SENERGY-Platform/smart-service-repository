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

func TestReleaseSearch(t *testing.T) {
	if CI {
		t.Skip("not in ci")
	}
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	apiUrl, _, err := apiTestEnv(ctx, wg, true, nil, func(err error) {
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
			BpmnXml:     resources.ComplexSelectionBpmn,
			SvgXml:      resources.ComplexSelectionSvg,
			Description: "test description",
			Name:        "test name",
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

	releases := []model.SmartServiceRelease{
		{
			Name:        "foo bar batz",
			Description: "unrelated",
			DesignId:    design.Id,
		},
		{
			Name:        "foo",
			Description: "bar",
			DesignId:    design.Id,
		},
		{
			Name:        "42",
			Description: "batz something",
			DesignId:    design.Id,
		},
		{
			Name:        "bar",
			Description: "something 42",
			DesignId:    design.Id,
		},
	}

	t.Run("create releases", func(t *testing.T) {
		for _, release := range releases {
			resp, err := post(userToken, apiUrl+"/releases", release)
			if err != nil {
				t.Error(err)
				return
			}
			if resp.StatusCode != http.StatusOK {
				temp, _ := io.ReadAll(resp.Body)
				t.Error(resp.StatusCode, string(temp))
				return
			}
			time.Sleep(1 * time.Second)
		}
	})

	expectedSearchCount := map[string]int{
		"foo":     2,
		"foo bar": 2, //finds elements that contain foo and bar
		"bar":     3,
		"42":      2,
		"batz":    2,
	}

	time.Sleep(2 * time.Second)

	for key, val := range expectedSearchCount {
		t.Run("asc search "+key, func(t *testing.T) {
			resp, err := get(userToken, apiUrl+"/releases?sort=name.asc&search="+url.QueryEscape(key))
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
			result := []model.SmartServiceDesign{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			if err != nil {
				t.Error(err)
				return
			}
			if len(result) != val {
				t.Error(key, val, len(result), result)
			}
		})
	}

	for key, val := range expectedSearchCount {
		t.Run("desc search "+key, func(t *testing.T) {
			resp, err := get(userToken, apiUrl+"/releases?sort=name.desc&search="+url.QueryEscape(key))
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
			result := []model.SmartServiceReleaseExtended{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			if err != nil {
				t.Error(err)
				return
			}
			if len(result) != val {
				t.Error(key, val, len(result), result)
			}
		})
	}
}

func TestReleaseOptionsApi2(t *testing.T) {
	if CI {
		t.Skip("not in ci")
	}
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	apiUrl, _, err := apiTestEnv(ctx, wg, false, resources.SelectionsResponse2Obj, func(err error) {
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
			BpmnXml:     resources.ComplexSelectionBpmn,
			SvgXml:      resources.ComplexSelectionSvg,
			Description: "test description",
			Name:        "test name",
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
		if design.BpmnXml != resources.ComplexSelectionBpmn {
			t.Error(design.BpmnXml)
			return
		}
		if design.SvgXml != resources.ComplexSelectionSvg {
			t.Error(design.SvgXml)
			return
		}
		if design.Id == "" {
			t.Error(design.Id)
			return
		}
		if design.Description != "test description" {
			t.Error(design.Description)
			return
		}
		if design.Name != "test name" {
			t.Error(design.Name)
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

	time.Sleep(2 * time.Second)

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

		if !reflect.DeepEqual(resources.ExpectedParams2Obj, parameters) {
			temp1, _ := json.Marshal(parameters)
			temp2, _ := json.Marshal(resources.ExpectedParams2Obj)
			t.Error("\n", string(temp1), "\n", string(temp2))
		}
	})
}

func TestReleaseOptionsApi(t *testing.T) {
	if CI {
		t.Skip("not in ci")
	}
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	apiUrl, _, err := apiTestEnv(ctx, wg, false, nil, func(err error) {
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
			BpmnXml:     resources.ParamsBpmn,
			SvgXml:      resources.ParamsSvg,
			Description: "test description",
			Name:        "test name",
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
		if design.BpmnXml != resources.ParamsBpmn {
			t.Error(design.BpmnXml)
			return
		}
		if design.SvgXml != resources.ParamsSvg {
			t.Error(design.SvgXml)
			return
		}
		if design.Id == "" {
			t.Error(design.Id)
			return
		}
		if design.Description != "test description" {
			t.Error(design.Description)
			return
		}
		if design.Name != "test name" {
			t.Error(design.Name)
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

	time.Sleep(2 * time.Second)

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

		if !reflect.DeepEqual(resources.ExpectedParams1Obj, parameters) {
			temp1, _ := json.Marshal(parameters)
			temp2, _ := json.Marshal(resources.ExpectedParams1Obj)
			t.Error("\n", string(temp1), "\n", string(temp2))
		}
	})
}

func TestReleaseOptionsWithCharacteristicApi(t *testing.T) {
	if CI {
		t.Skip("not in ci")
	}
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	apiUrl, _, err := apiTestEnv(ctx, wg, false, resources.SelectionsResponse3Obj, func(err error) {
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
			BpmnXml:     resources.ParamsBpmn,
			SvgXml:      resources.ParamsSvg,
			Description: "test description",
			Name:        "test name",
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
		if design.BpmnXml != resources.ParamsBpmn {
			t.Error(design.BpmnXml)
			return
		}
		if design.SvgXml != resources.ParamsSvg {
			t.Error(design.SvgXml)
			return
		}
		if design.Id == "" {
			t.Error(design.Id)
			return
		}
		if design.Description != "test description" {
			t.Error(design.Description)
			return
		}
		if design.Name != "test name" {
			t.Error(design.Name)
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

	time.Sleep(2 * time.Second)

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

		if !reflect.DeepEqual(resources.ExpectedParams3Obj, parameters) {
			temp1, _ := json.Marshal(parameters)
			temp2, _ := json.Marshal(resources.ExpectedParams3Obj)
			t.Error("\n", string(temp1), "\n", string(temp2))
		}
	})
}

func TestReleaseApi(t *testing.T) {
	if CI {
		t.Skip("not in ci")
	}
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	apiUrl, _, err := apiTestEnv(ctx, wg, true, nil, func(err error) {
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
			BpmnXml: resources.NamedDescBpmn,
			SvgXml:  resources.NamedDescSvg,
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
		if design.BpmnXml != resources.NamedDescBpmn {
			t.Error(design.BpmnXml)
			return
		}
		if design.SvgXml != resources.NamedDescSvg {
			t.Error(design.SvgXml)
			return
		}
		if design.Id == "" {
			t.Error(design.Id)
			return
		}
		if design.Description != "test description" {
			t.Error(design.Description)
			return
		}
		if design.Name != "test name" {
			t.Error(design.Name)
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

	time.Sleep(10 * time.Second) //allow async cqrs

	t.Run("read release", func(t *testing.T) {
		resp, err := get(userToken, apiUrl+"/releases/"+url.PathEscape(release.Id))
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
		temp := model.SmartServiceRelease{}
		err = json.NewDecoder(resp.Body).Decode(&temp)
		if err != nil {
			t.Error(err)
			return
		}
		if temp.Error != "" {
			t.Error(temp.Error)
			return
		}
	})

	t.Run("delete release", func(t *testing.T) {
		resp, err := delete(userToken, apiUrl+"/releases/"+url.PathEscape(release.Id))
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

	names := []string{"a", "b", "c", "d", "e", "f"}
	t.Run("create list releases", func(t *testing.T) {
		for _, name := range names {
			resp, err := post(userToken, apiUrl+"/releases", model.SmartServiceRelease{
				DesignId:    design.Id,
				Name:        name,
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
			time.Sleep(2 * time.Second)
		}
	})

	time.Sleep(10 * time.Second) //allow async cqrs delete to play out

	t.Run("list", func(t *testing.T) {
		t.Run("default", func(t *testing.T) {
			testReleaseList(t, apiUrl, "", names, names[len(names)-1])
		})
		t.Run("sort=name.asc", func(t *testing.T) {
			testReleaseList(t, apiUrl, "?sort=name.asc", names, names[len(names)-1])
		})
		t.Run("sort=name.desc", func(t *testing.T) {
			testReleaseList(t, apiUrl, "?sort=name.desc", reverse(names), names[len(names)-1])
		})
		t.Run("limit and offset", func(t *testing.T) {
			testReleaseList(t, apiUrl, "?limit=2&offset=1", names[1:3], "")
		})
		t.Run("newest", func(t *testing.T) {
			testReleaseList(t, apiUrl, "?latest=true", names[len(names)-1:], "")
		})
	})
}

func TestReleaseUpdate(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	apiUrl, _, err := apiTestEnv(ctx, wg, true, nil, func(err error) {
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
	})

	t.Run("read instance no new release", func(t *testing.T) {
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

		if instance.NewReleaseId != "" {
			t.Error(instance.Name)
			return
		}
	})

	release2 := model.SmartServiceRelease{}
	t.Run("create release 2", func(t *testing.T) {
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
		err = json.NewDecoder(resp.Body).Decode(&release2)
		if err != nil {
			t.Error(err)
			return
		}
	})

	time.Sleep(10 * time.Second)

	t.Run("read release1 new release id release2", func(t *testing.T) {
		resp, err := get(userToken, apiUrl+"/releases/"+url.PathEscape(release.Id))
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
		temp := model.SmartServiceRelease{}
		err = json.NewDecoder(resp.Body).Decode(&temp)
		if err != nil {
			t.Error(err)
			return
		}
		if temp.NewReleaseId != release2.Id {
			t.Error(temp.NewReleaseId)
			return
		}
	})

	t.Run("read release2 no new release id", func(t *testing.T) {
		resp, err := get(userToken, apiUrl+"/releases/"+url.PathEscape(release2.Id))
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
		temp := model.SmartServiceRelease{}
		err = json.NewDecoder(resp.Body).Decode(&temp)
		if err != nil {
			t.Error(err)
			return
		}
		if temp.NewReleaseId != "" {
			t.Error(temp.NewReleaseId)
			return
		}
	})

	t.Run("read instance new release 2", func(t *testing.T) {
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

		if instance.NewReleaseId != release2.Id {
			t.Error(instance.NewReleaseId, release.Id, release2.Id)
			return
		}
	})

	release3 := model.SmartServiceRelease{}
	t.Run("create release 3", func(t *testing.T) {
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
		err = json.NewDecoder(resp.Body).Decode(&release3)
		if err != nil {
			t.Error(err)
			return
		}
	})

	time.Sleep(10 * time.Second)

	t.Run("read release1 new release id release3", func(t *testing.T) {
		resp, err := get(userToken, apiUrl+"/releases/"+url.PathEscape(release.Id))
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
		temp := model.SmartServiceRelease{}
		err = json.NewDecoder(resp.Body).Decode(&temp)
		if err != nil {
			t.Error(err)
			return
		}
		if temp.NewReleaseId != release3.Id {
			t.Error(temp.NewReleaseId)
			return
		}
	})

	t.Run("read release2 new release id release3", func(t *testing.T) {
		resp, err := get(userToken, apiUrl+"/releases/"+url.PathEscape(release2.Id))
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
		temp := model.SmartServiceRelease{}
		err = json.NewDecoder(resp.Body).Decode(&temp)
		if err != nil {
			t.Error(err)
			return
		}
		if temp.NewReleaseId != release3.Id {
			t.Error(temp.NewReleaseId)
			return
		}
	})

	t.Run("read release3 no new release id", func(t *testing.T) {
		resp, err := get(userToken, apiUrl+"/releases/"+url.PathEscape(release3.Id))
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
		temp := model.SmartServiceRelease{}
		err = json.NewDecoder(resp.Body).Decode(&temp)
		if err != nil {
			t.Error(err)
			return
		}
		if temp.NewReleaseId != "" {
			t.Error(temp.NewReleaseId)
			return
		}
	})

	t.Run("read instance new release 3", func(t *testing.T) {
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

		if instance.NewReleaseId != release3.Id {
			t.Error(instance.NewReleaseId, release.Id, release2.Id, release3.Id)
			return
		}
	})

}

func testReleaseList(t *testing.T, apiUrl string, query string, expectedNamesOrder []string, newestName string) {
	resp, err := get(userToken, apiUrl+"/releases"+query)
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
	result := []model.SmartServiceRelease{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		t.Error(err)
		return
	}
	if len(result) != len(expectedNamesOrder) {
		t.Error(len(result), len(expectedNamesOrder), result)
		return
	}
	newest := model.SmartServiceRelease{}
	for i, element := range result {
		if element.Name == newestName {
			newest = element
		}
		if element.Name != expectedNamesOrder[i] {
			t.Error(element.Name, expectedNamesOrder[i])
			return
		}
	}
	if newestName != "" {
		for _, element := range result {
			if element.Id == newest.Id && element.NewReleaseId != "" {
				t.Error(element)
				return
			}
			if element.Id != newest.Id && element.NewReleaseId != newest.Id {
				t.Error("old:", element.NewReleaseId, "newest:", newest.Id, element)
				elementJson, _ := json.Marshal(element)
				newestJson, _ := json.Marshal(newest)
				t.Error(string(elementJson), "\n", string(newestJson))
				return
			}
		}
	}

}
