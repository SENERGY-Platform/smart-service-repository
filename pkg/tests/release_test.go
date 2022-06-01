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
	"runtime/debug"
	"sync"
	"testing"
	"time"
)

func TestReleaseApi(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	apiUrl, err := apiTestEnv(ctx, wg, true, func(err error) {
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
		err = json.NewDecoder(resp.Body).Decode(&release)
		if err != nil {
			t.Error(err)
			return
		}
	})

	time.Sleep(5 * time.Second) //allow async cqrs

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

	time.Sleep(2 * time.Second) //allow async cqrs delete to play out
}
