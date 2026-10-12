/*
 * Copyright 2026 InfAI (CC SES)
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
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/SENERGY-Platform/smart-service-repository/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/camunda"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/controller"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/database/mongo"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/tests/resources"
)

func TestReleaseResourceCheck(t *testing.T) {
	t.Setenv("DELETE_UNUSED_OLD_VERSION_RELEASES", "false")
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	apiUrl, config, _, perm, err := apiTestEnvWithPermClient(ctx, wg, false, nil, func(err error) {
		debug.PrintStack()
		t.Error(err)
	})
	if err != nil {
		t.Error(err)
		return
	}

	// the repositories: the process model was updated after the release, the flow before, the import type changes between two runs
	processModelUpdated := time.Now().Add(time.Hour).Unix()
	flowUpdated := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	var importTypeVersion atomic.Int64
	importTypeVersion.Store(1)
	var requestCount atomic.Int64
	repo := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestCount.Add(1)
		if request.Header.Get("Authorization") == "" {
			writer.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch request.URL.Path {
		case "/processes/pm-check":
			_ = json.NewEncoder(writer).Encode(map[string]interface{}{"id": "pm-check", "last_updated_unix": processModelUpdated})
		case "/flow/flow-check":
			_ = json.NewEncoder(writer).Encode(map[string]interface{}{"id": "flow-check", "dateUpdated": flowUpdated})
		case "/import-types/it-check":
			_ = json.NewEncoder(writer).Encode(map[string]interface{}{"id": "it-check", "version": importTypeVersion.Load()})
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer repo.Close()
	config.ProcessModelRepoUrl = repo.URL
	config.FlowRepoUrl = repo.URL
	config.ImportRepoUrl = repo.URL

	db, err := mongo.New(config)
	if err != nil {
		t.Error(err)
		return
	}
	tokenProvider, err := auth.GetCachedTokenProvider(config)
	if err != nil {
		t.Error(err)
		return
	}
	ctrl, err := controller.New(ctx, config, db, perm, camunda.New(config), nil, tokenProvider, nil)
	if err != nil {
		t.Error(err)
		return
	}

	postJson := func(path string, in interface{}, out interface{}) {
		t.Helper()
		resp, err := post(userToken, apiUrl+path, in)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			temp, _ := io.ReadAll(resp.Body)
			t.Fatal(resp.StatusCode, string(temp))
		}
		if err = json.NewDecoder(resp.Body).Decode(out); err != nil {
			t.Fatal(err)
		}
	}
	getBody := func(path string) string {
		t.Helper()
		resp, err := get(userToken, apiUrl+path)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		temp, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			t.Fatal(path, resp.StatusCode, string(temp))
		}
		return string(temp)
	}

	design := model.SmartServiceDesign{}
	postJson("/designs", model.SmartServiceDesign{
		Name:    "resource check test",
		BpmnXml: usageBpmn("pm-check", "flow-check", "it-check"),
		SvgXml:  resources.ProcessDeploymentSvg,
	}, &design)
	release := model.SmartServiceRelease{}
	postJson("/releases", model.SmartServiceRelease{
		DesignId:         design.Id,
		Name:             "resource check test",
		ChangedResources: []model.ChangedResource{{Kind: "flow", Id: "client-supplied", Reason: "gone", At: 1}}, // must be ignored
	}, &release)
	if len(release.ChangedResources) != 0 {
		t.Errorf("client supplied changed resources were accepted: %#v", release.ChangedResources)
	}

	result, err := ctrl.CheckReleaseResources(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result.Checked != 1 || result.Marked != 1 || result.Errors != 0 {
		t.Errorf("first run: %#v", result)
	}
	importTypeVersion.Store(2)
	result, err = ctrl.CheckReleaseResources(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result.Checked != 1 || result.Marked != 1 || result.Errors != 0 {
		t.Errorf("second run: %#v", result)
	}

	t.Run("the release lists the changed resources", func(t *testing.T) {
		got := model.SmartServiceRelease{}
		if err := json.Unmarshal([]byte(getBody("/releases/"+release.Id)), &got); err != nil {
			t.Fatal(err)
		}
		if len(got.ChangedResources) != 2 {
			t.Fatalf("%#v", got.ChangedResources)
		}
		if first := got.ChangedResources[0]; first != (model.ChangedResource{Kind: "process_model", Id: "pm-check", Reason: "changed", At: processModelUpdated}) {
			t.Errorf("%#v", first)
		}
		second := got.ChangedResources[1]
		if second.Kind != "import_type" || second.Id != "it-check" || second.Reason != "changed" || second.At < release.CreatedAt {
			t.Errorf("%#v", second)
		}

		list := []model.SmartServiceRelease{}
		if err := json.Unmarshal([]byte(getBody("/releases")), &list); err != nil {
			t.Fatal(err)
		}
		if len(list) != 1 || !reflect.DeepEqual(list[0].ChangedResources, got.ChangedResources) {
			t.Errorf("%#v", list)
		}
	})

	t.Run("the import type baseline is stored but served nowhere", func(t *testing.T) {
		stored, err, _ := db.GetRelease(ctx, release.Id, false)
		if err != nil {
			t.Fatal(err)
		}
		if len(stored.ImportTypeHashes) != 1 || stored.ImportTypeHashes["it-check"] == "" {
			t.Fatalf("%#v", stored.ImportTypeHashes)
		}
		for _, path := range []string{"/releases/" + release.Id, "/releases", "/extended-releases/" + release.Id, "/extended-releases"} {
			body := getBody(path)
			if strings.Contains(body, "import_type_hashes") || strings.Contains(body, stored.ImportTypeHashes["it-check"]) {
				t.Errorf("%v serves the import type baseline: %v", path, body)
			}
			if !strings.Contains(body, "changed_resources") && strings.HasPrefix(path, "/releases") {
				t.Errorf("%v: missing changed_resources", path)
			}
		}
	})

	t.Run("the check leaves the rest of the release untouched and a second run writes nothing new", func(t *testing.T) {
		before, err, _ := db.GetRelease(ctx, release.Id, false)
		if err != nil {
			t.Fatal(err)
		}
		requestsBefore := requestCount.Load()
		_, err = ctrl.CheckReleaseResources(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if requestCount.Load() != requestsBefore+3 {
			t.Errorf("expected 3 requests per run, got %v", requestCount.Load()-requestsBefore)
		}
		after, err, _ := db.GetRelease(ctx, release.Id, false)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(before, after) {
			t.Errorf("\nbefore %#v\nafter  %#v", before, after)
		}
		if after.UsedResources == nil || after.BpmnXml == "" {
			t.Error("release lost fields")
		}
	})
}
