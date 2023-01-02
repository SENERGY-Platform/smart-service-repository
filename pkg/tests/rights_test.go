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
	"github.com/SENERGY-Platform/smart-service-repository/pkg/permissions"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/tests/resources"
	"io"
	"net/http"
	"reflect"
	"runtime/debug"
	"sync"
	"testing"
	"time"
)

func TestReleaseRights(t *testing.T) {
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

	t.Run("read release1 rights", func(t *testing.T) {
		rights, err, _ := permissions.New(config).GetResourceRights(adminToken, config.KafkaSmartServiceReleaseTopic, release.Id)
		if err != nil {
			t.Error(err)
			return
		}
		if !reflect.DeepEqual(rights.UserRights, map[string]permissions.Right{
			userId: {
				Read:         true,
				Write:        true,
				Execute:      true,
				Administrate: true,
			},
		}) {
			t.Error(rights.UserRights)
			return
		}
	})

	t.Run("set release 1 right", func(t *testing.T) {
		err = permissions.New(config).SetResourceRights(adminToken, config.KafkaSmartServiceReleaseTopic, release.Id, permissions.ResourceRights{
			UserRights: map[string]permissions.Right{
				userId: {
					Read:         true,
					Write:        true,
					Execute:      true,
					Administrate: true,
				},
				adminId: {
					Read:         true,
					Write:        true,
					Execute:      true,
					Administrate: true,
				},
			},
			GroupRights: map[string]permissions.Right{},
		}, release.DesignId+"/"+release.Id+"_"+"rights")
		if err != nil {
			t.Error(err)
			return
		}
	})

	time.Sleep(5 * time.Second) //allow async cqrs

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

	t.Run("read release2 rights", func(t *testing.T) {
		rights, err, _ := permissions.New(config).GetResourceRights(adminToken, config.KafkaSmartServiceReleaseTopic, release2.Id)
		if err != nil {
			t.Error(err)
			return
		}
		if !reflect.DeepEqual(rights.UserRights, map[string]permissions.Right{
			userId: {
				Read:         true,
				Write:        true,
				Execute:      true,
				Administrate: true,
			},
			adminId: {
				Read:         true,
				Write:        true,
				Execute:      true,
				Administrate: true,
			},
		}) {
			t.Error(rights.UserRights)
			return
		}
	})

}
