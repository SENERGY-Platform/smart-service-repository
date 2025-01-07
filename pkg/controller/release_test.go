/*
 * Copyright 2024 InfAI (CC SES)
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

package controller

import (
	"context"
	"errors"
	devicerepository "github.com/SENERGY-Platform/device-repository/lib/client"
	permclient "github.com/SENERGY-Platform/permissions-v2/pkg/client"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/database/mongo"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/selectables"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/tests/docker"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/tests/mocks"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/tests/resources"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestReleaseDeleteRetry(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config, err := configuration.Load("../../config.json")
	if err != nil {
		t.Error(err)
		return
	}

	config.MarkAgeLimit.SetDuration(time.Millisecond)

	_, mongoIp, err := docker.MongoDB(ctx, wg)
	if err != nil {
		t.Error(err)
		return
	}
	config.MongoUrl = "mongodb://" + mongoIp + ":27017"

	_, permV2Ip, err := docker.PermissionsV2(ctx, wg, config.MongoUrl)
	if err != nil {
		t.Error(err)
		return
	}
	config.PermissionsV2Url = "http://" + permV2Ip + ":8080"

	tokenprovider, err := auth.GetCachedTokenProvider(config)
	if err != nil {
		t.Error(err)
		return
	}

	db, err := mongo.New(config)
	if err != nil {
		t.Error(err)
		return
	}

	permClient := permclient.New(config.PermissionsV2Url)

	camundaMock := &mocks.CamundaErrMock{Err: nil}

	cmd, err := New(
		ctx,
		config,
		db,
		permClient,
		camundaMock,
		selectables.New(config),
		tokenprovider,
		devicerepository.NewClient(config.DeviceRepositoryUrl, nil),
	)
	if err != nil {
		t.Error(err)
		return
	}

	err = cmd.saveReleaseCreate(model.SmartServiceReleaseExtended{
		SmartServiceRelease: model.SmartServiceRelease{
			Id:        "test-release-id-1",
			DesignId:  "test-design-id-1",
			Name:      "name-1",
			CreatedAt: time.Now().UnixMilli(),
			Creator:   "test-creator-1",
		},
		BpmnXml: resources.ProcessDeploymentBpmn,
		SvgXml:  resources.ProcessDeploymentSvg,
	})
	if err != nil {
		t.Error(err)
		return
	}

	camundaMock.Err = errors.New("test-error")

	err = cmd.deleteRelease("test-release-id-1")
	if err == nil || !errors.Is(err, camundaMock.Err) {
		t.Error(err)
		return
	}

	time.Sleep(10 * time.Millisecond)

	_, err, _ = db.GetRelease("test-release-id-1", true)
	if err != nil {
		t.Error(err)
		return
	}
	todelete, unfinished, err := db.GetMarkedReleases()
	if err != nil {
		t.Error(err)
		return
	}
	if len(unfinished) != 0 {
		t.Error(unfinished)
	}
	if len(todelete) != 1 {
		t.Error(todelete)
	}

	cmd.retryMarkedReleases()

	_, err, _ = db.GetRelease("test-release-id-1", true)
	if err != nil {
		t.Error(err)
		return
	}
	todelete, unfinished, err = db.GetMarkedReleases()
	if err != nil {
		t.Error(err)
		return
	}
	if len(unfinished) != 0 {
		t.Error(unfinished)
	}
	if len(todelete) != 1 {
		t.Error(todelete)
	}

	camundaMock.Err = nil

	cmd.retryMarkedReleases()

	_, err, _ = db.GetRelease("test-release-id-1", true)
	if err == nil {
		t.Error(err)
		return
	}
	todelete, unfinished, err = db.GetMarkedReleases()
	if err != nil {
		t.Error(err)
		return
	}
	if len(unfinished) != 0 {
		t.Error(unfinished)
	}
	if len(todelete) != 0 {
		t.Error(todelete)
	}

	_, _, code := permClient.GetResource(permclient.InternalAdminToken, config.SmartServiceReleasePermissionsTopic, "test-release-id-1")
	if code != http.StatusNotFound {
		t.Error(code)
		return
	}
}

func TestUnfinishedReleaseRollback(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config, err := configuration.Load("../../config.json")
	if err != nil {
		t.Error(err)
		return
	}

	config.MarkAgeLimit.SetDuration(time.Millisecond)

	_, mongoIp, err := docker.MongoDB(ctx, wg)
	if err != nil {
		t.Error(err)
		return
	}
	config.MongoUrl = "mongodb://" + mongoIp + ":27017"

	_, permV2Ip, err := docker.PermissionsV2(ctx, wg, config.MongoUrl)
	if err != nil {
		t.Error(err)
		return
	}
	config.PermissionsV2Url = "http://" + permV2Ip + ":8080"

	tokenprovider, err := auth.GetCachedTokenProvider(config)
	if err != nil {
		t.Error(err)
		return
	}

	db, err := mongo.New(config)
	if err != nil {
		t.Error(err)
		return
	}

	permClient := permclient.New(config.PermissionsV2Url)

	camundaMock := &mocks.CamundaErrMock{Err: errors.New("test-error")}

	cmd, err := New(
		ctx,
		config,
		db,
		permClient,
		camundaMock,
		selectables.New(config),
		tokenprovider,
		devicerepository.NewClient(config.DeviceRepositoryUrl, nil),
	)
	if err != nil {
		t.Error(err)
		return
	}

	release := model.SmartServiceReleaseExtended{
		SmartServiceRelease: model.SmartServiceRelease{
			Id:        "test-release-id-1",
			DesignId:  "test-design-id-1",
			Name:      "name-1",
			CreatedAt: time.Now().UnixMilli(),
			Creator:   "test-creator-1",
		},
		BpmnXml: resources.ProcessDeploymentBpmn,
		SvgXml:  resources.ProcessDeploymentSvg,
	}

	err, _ = db.SetRelease(release, true)
	if err != nil {
		t.Error(err)
		return
	}
	err = cmd.deployRelease(release, []model.SmartServiceReleaseExtended{})
	if !errors.Is(err, camundaMock.Err) || err == nil {
		t.Error(err)
		return
	}

	time.Sleep(10 * time.Millisecond)

	_, err, _ = db.GetRelease("test-release-id-1", true)
	if err != nil {
		t.Error(err)
		return
	}
	todelete, unfinished, err := db.GetMarkedReleases()
	if err != nil {
		t.Error(err)
		return
	}
	if len(unfinished) != 1 {
		t.Error(unfinished, todelete)
		return
	}
	if len(todelete) != 0 {
		t.Error(unfinished, todelete)
		return
	}

	cmd.retryMarkedReleases()

	_, err, _ = db.GetRelease("test-release-id-1", true)
	if err != nil {
		t.Error(err)
		return
	}
	todelete, unfinished, err = db.GetMarkedReleases()
	if err != nil {
		t.Error(err)
		return
	}
	if len(unfinished) != 0 {
		t.Error(unfinished)
		return
	}
	if len(todelete) != 1 {
		t.Error(todelete)
		return
	}

	camundaMock.Err = nil

	cmd.retryMarkedReleases()

	_, err, _ = db.GetRelease("test-release-id-1", true)
	if err == nil {
		t.Error(err)
		return
	}
	todelete, unfinished, err = db.GetMarkedReleases()
	if err != nil {
		t.Error(err)
		return
	}
	if len(unfinished) != 0 {
		t.Error(unfinished)
	}
	if len(todelete) != 0 {
		t.Error(todelete)
	}

	_, _, code := permClient.GetResource(permclient.InternalAdminToken, config.SmartServiceReleasePermissionsTopic, "test-release-id-1")
	if code != http.StatusNotFound {
		t.Error(code)
		return
	}
}

func TestReleaseDeploymentRollback(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config, err := configuration.Load("../../config.json")
	if err != nil {
		t.Error(err)
		return
	}

	config.MarkAgeLimit.SetDuration(time.Millisecond)

	_, mongoIp, err := docker.MongoDB(ctx, wg)
	if err != nil {
		t.Error(err)
		return
	}
	config.MongoUrl = "mongodb://" + mongoIp + ":27017"

	_, permV2Ip, err := docker.PermissionsV2(ctx, wg, config.MongoUrl)
	if err != nil {
		t.Error(err)
		return
	}
	config.PermissionsV2Url = "http://" + permV2Ip + ":8080"

	tokenprovider, err := auth.GetCachedTokenProvider(config)
	if err != nil {
		t.Error(err)
		return
	}

	db, err := mongo.New(config)
	if err != nil {
		t.Error(err)
		return
	}

	permClient := permclient.New(config.PermissionsV2Url)

	camundaMock := &mocks.CamundaErrMock{Err: errors.New("test-error")}

	cmd, err := New(
		ctx,
		config,
		db,
		permClient,
		camundaMock,
		selectables.New(config),
		tokenprovider,
		devicerepository.NewClient(config.DeviceRepositoryUrl, nil),
	)
	if err != nil {
		t.Error(err)
		return
	}

	err = cmd.saveReleaseCreate(model.SmartServiceReleaseExtended{
		SmartServiceRelease: model.SmartServiceRelease{
			Id:        "test-release-id-1",
			DesignId:  "test-design-id-1",
			Name:      "name-1",
			CreatedAt: time.Now().UnixMilli(),
			Creator:   "test-creator-1",
		},
		BpmnXml: resources.ProcessDeploymentBpmn,
		SvgXml:  resources.ProcessDeploymentSvg,
	})
	if !errors.Is(err, camundaMock.Err) {
		t.Error(err)
		return
	}

	time.Sleep(10 * time.Millisecond)

	_, err, _ = db.GetRelease("test-release-id-1", true)
	if err != nil {
		t.Error(err)
		return
	}
	todelete, unfinished, err := db.GetMarkedReleases()
	if err != nil {
		t.Error(err)
		return
	}
	if len(unfinished) != 0 {
		t.Error(unfinished)
	}
	if len(todelete) != 1 {
		t.Error(todelete)
	}

	cmd.retryMarkedReleases()

	_, err, _ = db.GetRelease("test-release-id-1", true)
	if err != nil {
		t.Error(err)
		return
	}
	todelete, unfinished, err = db.GetMarkedReleases()
	if err != nil {
		t.Error(err)
		return
	}
	if len(unfinished) != 0 {
		t.Error(unfinished)
	}
	if len(todelete) != 1 {
		t.Error(todelete)
	}

	camundaMock.Err = nil

	cmd.retryMarkedReleases()

	_, err, _ = db.GetRelease("test-release-id-1", true)
	if err == nil {
		t.Error(err)
		return
	}
	todelete, unfinished, err = db.GetMarkedReleases()
	if err != nil {
		t.Error(err)
		return
	}
	if len(unfinished) != 0 {
		t.Error(unfinished)
	}
	if len(todelete) != 0 {
		t.Error(todelete)
	}

	_, _, code := permClient.GetResource(permclient.InternalAdminToken, config.SmartServiceReleasePermissionsTopic, "test-release-id-1")
	if code != http.StatusNotFound {
		t.Error(code)
		return
	}
}
