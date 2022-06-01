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
)

func TestDesignApi(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	apiUrl, _, err := apiTestEnv(ctx, wg, false, func(err error) {
		debug.PrintStack()
		t.Error(err)
	})
	if err != nil {
		t.Error(err)
		return
	}

	design := model.SmartServiceDesign{}
	t.Run("create", func(t *testing.T) {
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
	t.Run("read after create", func(t *testing.T) {
		testDesignRead(t, apiUrl, design)
	})
	t.Run("update", func(t *testing.T) {
		design.Name = "changed name"
		resp, err := put(userToken, apiUrl+"/designs/"+url.PathEscape(design.Id), design)
		if err != nil {
			t.Error(err)
			return
		}
		if resp.StatusCode != http.StatusOK {
			temp, _ := io.ReadAll(resp.Body)
			t.Error(resp.StatusCode, string(temp))
			return
		}
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
		if design.Name != "changed name" {
			t.Error(design.Name)
			return
		}
	})
	t.Run("read after update", func(t *testing.T) {
		testDesignRead(t, apiUrl, design)
	})

	names := []string{"a", "b", "c", "d", "e", "f"}
	t.Run("create list", func(t *testing.T) {
		for _, name := range names {
			resp, err := post(userToken, apiUrl+"/designs", model.SmartServiceDesign{
				BpmnXml: resources.NamedDescBpmn,
				SvgXml:  resources.NamedDescSvg,
				Name:    name,
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
		}
	})

	t.Run("delete", func(t *testing.T) {
		resp, err := delete(userToken, apiUrl+"/designs/"+url.PathEscape(design.Id))
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
			testDesignList(t, apiUrl, "", names)
		})
		t.Run("sort=name.asc", func(t *testing.T) {
			testDesignList(t, apiUrl, "?sort=name.asc", names)
		})
		t.Run("sort=name.desc", func(t *testing.T) {
			testDesignList(t, apiUrl, "?sort=name.desc", reverse(names))
		})
		t.Run("limit and offset", func(t *testing.T) {
			testDesignList(t, apiUrl, "?limit=2&offset=1", names[1:3])
		})
	})

}

func testDesignList(t *testing.T, apiUrl string, query string, expectedNamesOrder []string) {
	resp, err := get(userToken, apiUrl+"/designs"+query)
	if err != nil {
		t.Error(err)
		return
	}
	if resp.StatusCode != http.StatusOK {
		temp, _ := io.ReadAll(resp.Body)
		t.Error(resp.StatusCode, string(temp))
		return
	}
	result := []model.SmartServiceDesign{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		t.Error(err)
		return
	}
	if len(result) != len(expectedNamesOrder) {
		t.Error(len(result), len(expectedNamesOrder), result)
		return
	}
	for i, element := range result {
		if element.Name != expectedNamesOrder[i] {
			t.Error(element.Name, expectedNamesOrder[i])
			return
		}
	}
}

func testDesignRead(t *testing.T, apiUrl string, design model.SmartServiceDesign) {
	resp, err := get(userToken, apiUrl+"/designs/"+url.PathEscape(design.Id))
	if err != nil {
		t.Error(err)
		return
	}
	if resp.StatusCode != http.StatusOK {
		temp, _ := io.ReadAll(resp.Body)
		t.Error(resp.StatusCode, string(temp))
		return
	}
	temp := model.SmartServiceDesign{}
	err = json.NewDecoder(resp.Body).Decode(&temp)
	if err != nil {
		t.Error(err)
		return
	}
	if !reflect.DeepEqual(design, temp) {
		t.Error(temp)
		return
	}
}
