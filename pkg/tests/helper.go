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
	devicerepository "github.com/SENERGY-Platform/device-repository/lib/client"
	"github.com/SENERGY-Platform/device-repository/lib/database"
	permissionsv2 "github.com/SENERGY-Platform/permissions-v2/pkg/client"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/api"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/camunda"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/controller"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/database/mongo"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/tests/docker"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/tests/mocks"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/tests/resources"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"runtime/debug"
	"strings"
	"sync"
	"testing"
	"time"
)

func apiTestEnv(ctx context.Context, wg *sync.WaitGroup, camundaAndCqrsDependencies bool, selectionResp []model.Selectable, errHandler func(error)) (apiUrl string, config configuration.Config, devicerepoTestDb database.Database, err error) {
	apiUrl, config, devicerepoTestDb, _, err = apiTestEnvWithPermClient(ctx, wg, camundaAndCqrsDependencies, selectionResp, errHandler)
	return
}

func apiTestEnvWithPermClient(ctx context.Context, wg *sync.WaitGroup, camundaAndCqrsDependencies bool, selectionResp []model.Selectable, errHandler func(error)) (apiUrl string, config configuration.Config, devicerepoTestDb database.Database, perm permissionsv2.Client, err error) {
	if selectionResp == nil {
		selectionResp = resources.SelectionsResponse1Obj
	}

	config, err = configuration.Load("../../config.json")
	if err != nil {
		return "", config, devicerepoTestDb, perm, err
	}
	config.Debug = true

	notificationMock := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		msg, _ := io.ReadAll(request.Body)
		log.Println("NOTIFICATION:", string(msg))
	}))
	wg.Add(1)
	go func() {
		<-ctx.Done()
		notificationMock.Close()
		wg.Done()
	}()
	config.NotificationUrl = notificationMock.URL

	port, _, err := docker.MongoDB(ctx, wg)
	if err != nil {
		debug.PrintStack()
		return "", config, devicerepoTestDb, perm, err
	}
	config.MongoUrl = "mongodb://localhost:" + port
	config.MongoWithTransactions = false

	db, err := mongo.New(config)
	if err != nil {
		return "", config, devicerepoTestDb, perm, err
	}

	devicerepo, devicerepoTestDb, err := devicerepository.NewTestClient()
	if err != nil {
		return "", config, devicerepoTestDb, perm, err
	}
	perm, err = permissionsv2.NewTestClient(ctx)
	if err != nil {
		return "", config, devicerepoTestDb, perm, err
	}

	if camundaAndCqrsDependencies {
		_, camundaPgIp, _, err := docker.Postgres(ctx, wg, "camunda")
		if err != nil {
			return "", config, devicerepoTestDb, perm, err
		}

		config.CamundaUrl, err = docker.Camunda(ctx, wg, camundaPgIp, "5432")
		if err != nil {
			return "", config, devicerepoTestDb, perm, err
		}
		time.Sleep(5 * time.Second)
	} else {
		camundaMock := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			element := map[string]interface{}{"id": "foobar", "deploymentId": "foo"}
			if strings.HasSuffix(request.URL.Path, "process-definition") {
				json.NewEncoder(writer).Encode([]interface{}{element})
			} else {
				json.NewEncoder(writer).Encode(element)
			}
		}))
		wg.Add(1)
		go func() {
			<-ctx.Done()
			camundaMock.Close()
			wg.Done()
		}()
		config.CamundaUrl = camundaMock.URL

	}

	selectablesMock := mocks.NewSelectables(selectionResp)

	config.AuthEndpoint = mocks.Keycloak(ctx, wg)

	tokenprovider, err := auth.GetCachedTokenProvider(config)
	if err != nil {
		return "", config, devicerepoTestDb, perm, err
	}

	ctrl, err := controller.New(ctx, config, db, perm, camunda.New(config), selectablesMock, tokenprovider, devicerepo)
	if err != nil {
		return "", config, devicerepoTestDb, perm, err
	}

	router := api.GetRouter(config, ctrl)
	server := httptest.NewServer(router)
	wg.Add(1)
	go func() {
		<-ctx.Done()
		server.Close()
		wg.Done()
	}()
	return server.URL, config, devicerepoTestDb, perm, nil
}

var SleepAfterEdit = 0 * time.Second

const userToken = "Bearer eyJhbGciOiJSUzI1NiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICIzaUtabW9aUHpsMmRtQnBJdS1vSkY4ZVVUZHh4OUFIckVOcG5CcHM5SjYwIn0.eyJqdGkiOiIzMmE1OTljZC0zNDgxLTQzYWUtYWY0NC04YTVmNjU4NzYxZTUiLCJleHAiOjE1NjI5MjAwMDUsIm5iZiI6MCwiaWF0IjoxNTYyOTE2NDA1LCJpc3MiOiJodHRwczovL2F1dGguc2VwbC5pbmZhaS5vcmcvYXV0aC9yZWFsbXMvbWFzdGVyIiwiYXVkIjoiZnJvbnRlbmQiLCJzdWIiOiJlYmJhZDkyNy00YzM5LTRkMTItODY5MC04OWIwNjdkZDRjZTciLCJ0eXAiOiJCZWFyZXIiLCJhenAiOiJmcm9udGVuZCIsIm5vbmNlIjoiNTVlMzA4N2UtZjljNi00MmQ2LWE0MmEtMGZiMjcxNWE4OTkyIiwiYXV0aF90aW1lIjoxNTYyOTE2NDA0LCJzZXNzaW9uX3N0YXRlIjoiYmU5MDQ2MmYtOGE3Yy00NWU4LTg1MjAtMGRlYzViZWI1ZWZlIiwiYWNyIjoiMSIsImFsbG93ZWQtb3JpZ2lucyI6WyIqIl0sInJlYWxtX2FjY2VzcyI6eyJyb2xlcyI6WyJ1bWFfYXV0aG9yaXphdGlvbiIsInVzZXIiXX0sInJlc291cmNlX2FjY2VzcyI6eyJhY2NvdW50Ijp7InJvbGVzIjpbIm1hbmFnZS1hY2NvdW50IiwibWFuYWdlLWFjY291bnQtbGlua3MiLCJ2aWV3LXByb2ZpbGUiXX19LCJyb2xlcyI6WyJ1bWFfYXV0aG9yaXphdGlvbiIsInVzZXIiLCJvZmZsaW5lX2FjY2VzcyJdLCJwcmVmZXJyZWRfdXNlcm5hbWUiOiJpbmdvIn0.pggKYb3V0VxFINWBqpFE_t14MKhSM7bhw8YqrYBRvOzh8ft7zu_-bOvLOYbJBwo0GU1D68U2d_eerkYEIt-mc0dNtdFasy5DG_GtvnWA4nsbf0BVsYKSZcRiDK4d4qbHu9NMjBdEwSkP9KDGEtou0yHtOnVzB1eHHNm_uSUO-O_kz2LWsXOPK2sbL1LTiCKS0XToJPdlaNczDMZB0nXR3sHbyi3Lwk-Va2ATS6Kke5M1KmFMowK-Y0jK2urt8GnCBIXvZMT6gUW9-dvlv4w_lAuVXQ9hFg_r0sBnoWzZOUR_xlrz2T-syjrZzmXlAkJrcD8KWPH-lCs0jD9pdiROhQ"
const userId = "ebbad927-4c39-4d12-8690-89b067dd4ce7"
const adminToken = "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJqdGkiOiIzMmE1OTljZC0zNDgxLTQzYWUtYWY0NC04YTVmNjU4NzYxZTUiLCJleHAiOjE1NjI5MjAwMDUsIm5iZiI6MCwiaWF0IjoxNTYyOTE2NDA1LCJpc3MiOiJodHRwczovL2F1dGguc2VwbC5pbmZhaS5vcmcvYXV0aC9yZWFsbXMvbWFzdGVyIiwiYXVkIjoiZnJvbnRlbmQiLCJzdWIiOiJhZG1pbklkIiwidHlwIjoiQmVhcmVyIiwiYXpwIjoiZnJvbnRlbmQiLCJub25jZSI6IjU1ZTMwODdlLWY5YzYtNDJkNi1hNDJhLTBmYjI3MTVhODk5MiIsImF1dGhfdGltZSI6MTU2MjkxNjQwNCwic2Vzc2lvbl9zdGF0ZSI6ImJlOTA0NjJmLThhN2MtNDVlOC04NTIwLTBkZWM1YmViNWVmZSIsImFjciI6IjEiLCJhbGxvd2VkLW9yaWdpbnMiOlsiKiJdLCJyZWFsbV9hY2Nlc3MiOnsicm9sZXMiOlsiYWRtaW4iLCJ1bWFfYXV0aG9yaXphdGlvbiIsInVzZXIiXX0sInJlc291cmNlX2FjY2VzcyI6eyJhY2NvdW50Ijp7InJvbGVzIjpbIm1hbmFnZS1hY2NvdW50IiwibWFuYWdlLWFjY291bnQtbGlua3MiLCJ2aWV3LXByb2ZpbGUiXX19LCJyb2xlcyI6WyJ1bWFfYXV0aG9yaXphdGlvbiIsInVzZXIiLCJvZmZsaW5lX2FjY2VzcyJdLCJwcmVmZXJyZWRfdXNlcm5hbWUiOiJpbmdvIn0.QjnVUFqPXxOdk0pCrZSvFWda6GumH_XFyrUlJ8H5Ysc"
const adminId = "adminId"

func get(token string, url string) (resp *http.Response, err error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	return
}

func put(token string, url string, msg interface{}) (resp *http.Response, err error) {
	body := new(bytes.Buffer)
	err = json.NewEncoder(body).Encode(msg)
	if err != nil {
		return resp, err
	}
	req, err := http.NewRequest("PUT", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if SleepAfterEdit != 0 {
		time.Sleep(SleepAfterEdit)
	}
	return
}

func post(token string, url string, msg interface{}) (resp *http.Response, err error) {
	body := new(bytes.Buffer)
	err = json.NewEncoder(body).Encode(msg)
	if err != nil {
		return resp, err
	}
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if SleepAfterEdit != 0 {
		time.Sleep(SleepAfterEdit)
	}
	return
}

func delete(token string, url string) (resp *http.Response, err error) {
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", token)
	resp, err = http.DefaultClient.Do(req)
	if SleepAfterEdit != 0 {
		time.Sleep(SleepAfterEdit)
	}
	return
}

func reverse[T any](s []T) (result []T) {
	for i := len(s) - 1; i >= 0; i-- {
		result = append(result, s[i])
	}
	return result
}

func checkContentType(t *testing.T, resp *http.Response) {
	t.Helper()
	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Error(contentType)
	}
}
