/*
 * Copyright (c) 2026 InfAI (CC SES)
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
	"net/url"
	"reflect"
	"runtime/debug"
	"sync"
	"testing"
	"time"

	permclient "github.com/SENERGY-Platform/permissions-v2/pkg/client"
	permmodel "github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/camunda"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/controller"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/database/mongo"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/tests/resources"
	"github.com/google/uuid"
)

// usageBpmn references one process model, flow and import type literally, and a second process model only by expression
func usageBpmn(processModelId string, flowId string, importTypeId string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:camunda="http://camunda.org/schema/1.0/bpmn" id="Definitions_1" targetNamespace="http://bpmn.io/schema/bpmn">
    <bpmn:process id="usage_test" name="usage test" isExecutable="true">
        <bpmn:startEvent id="StartEvent_1"/>
        <bpmn:serviceTask id="Task_process" camunda:type="external" camunda:topic="process_deployment">
            <bpmn:extensionElements><camunda:inputOutput>
                <camunda:inputParameter name="process_deployment.process_model_id">
                    ` + processModelId + `
                </camunda:inputParameter>
            </camunda:inputOutput></bpmn:extensionElements>
        </bpmn:serviceTask>
        <bpmn:serviceTask id="Task_process_expression" camunda:type="external" camunda:topic="process_deployment">
            <bpmn:extensionElements><camunda:inputOutput>
                <camunda:inputParameter name="process_deployment.process_model_id">${model_id}</camunda:inputParameter>
            </camunda:inputOutput></bpmn:extensionElements>
        </bpmn:serviceTask>
        <bpmn:serviceTask id="Task_analytics" camunda:type="external" camunda:topic="analytics">
            <bpmn:extensionElements><camunda:inputOutput>
                <camunda:inputParameter name="analytics.flow_id">` + flowId + `</camunda:inputParameter>
            </camunda:inputOutput></bpmn:extensionElements>
        </bpmn:serviceTask>
        <bpmn:serviceTask id="Task_import" camunda:type="external" camunda:topic="import">
            <bpmn:extensionElements><camunda:inputOutput>
                <camunda:inputParameter name="import.request">{"import_type_id": "` + importTypeId + `", "name": "${name}"}</camunda:inputParameter>
            </camunda:inputOutput></bpmn:extensionElements>
        </bpmn:serviceTask>
    </bpmn:process>
</bpmn:definitions>`
}

func TestResourceUsage(t *testing.T) {
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

	db, err := mongo.New(config)
	if err != nil {
		t.Error(err)
		return
	}

	createDesign := func(t *testing.T, bpmn string) (design model.SmartServiceDesign) {
		t.Helper()
		resp, err := post(userToken, apiUrl+"/designs", model.SmartServiceDesign{
			Name:    "usage test",
			BpmnXml: bpmn,
			SvgXml:  resources.ProcessDeploymentSvg,
		})
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			temp, _ := io.ReadAll(resp.Body)
			t.Fatal(resp.StatusCode, string(temp))
		}
		err = json.NewDecoder(resp.Body).Decode(&design)
		if err != nil {
			t.Fatal(err)
		}
		return design
	}

	createRelease := func(t *testing.T, design model.SmartServiceDesign, name string) (release model.SmartServiceRelease) {
		t.Helper()
		time.Sleep(1100 * time.Millisecond) // created_at has seconds resolution, and only an older release gets a new_release_id
		resp, err := post(userToken, apiUrl+"/releases", model.SmartServiceRelease{DesignId: design.Id, Name: name})
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			temp, _ := io.ReadAll(resp.Body)
			t.Fatal(resp.StatusCode, string(temp))
		}
		err = json.NewDecoder(resp.Body).Decode(&release)
		if err != nil {
			t.Fatal(err)
		}
		return release
	}

	createInstance := func(t *testing.T, instanceUserId string, release model.SmartServiceRelease) {
		t.Helper()
		err, _ := db.SetInstance(ctx, model.SmartServiceInstance{
			Id:        uuid.NewString(),
			UserId:    instanceUserId,
			DesignId:  release.DesignId,
			ReleaseId: release.Id,
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	// stores a release directly, without used_resources, like releases stored before the field existed
	storeLegacyRelease := func(t *testing.T, id string, designId string, bpmn string) {
		t.Helper()
		err, _ := db.SetRelease(ctx, model.SmartServiceReleaseExtended{
			SmartServiceRelease: model.SmartServiceRelease{
				Id:        id,
				DesignId:  designId,
				Name:      "legacy " + id,
				CreatedAt: time.Now().Unix(),
				Creator:   userId,
			},
			BpmnXml: bpmn,
			SvgXml:  resources.ProcessDeploymentSvg,
		}, false)
		if err != nil {
			t.Fatal(err)
		}
		stored, err, _ := db.GetRelease(ctx, id, true)
		if err != nil {
			t.Fatal(err)
		}
		if stored.UsedResources != nil {
			t.Fatal("legacy release stored with used resources", stored.UsedResources)
		}
	}

	getUsage := func(t *testing.T, token string, kind string, id string) (result model.ResourceUsage, code int) {
		t.Helper()
		resp, err := get(token, apiUrl+"/resource-usage/"+kind+"/"+url.PathEscape(id))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return result, resp.StatusCode
		}
		checkContentType(t, resp)
		temp, _ := io.ReadAll(resp.Body)
		// the body must carry exactly these keys, so nothing else about foreign releases leaks
		keys := map[string]interface{}{}
		err = json.Unmarshal(temp, &keys)
		if err != nil {
			t.Fatal(err)
		}
		if len(keys) != 3 || keys["releases"] == nil || keys["instances"] == nil || keys["readable"] == nil {
			t.Error("unexpected body", string(temp))
		}
		err = json.Unmarshal(temp, &result)
		if err != nil {
			t.Fatal(err)
		}
		return result, resp.StatusCode
	}

	expectUsage := func(t *testing.T, token string, kind string, id string, releases int64, instances int64, readable ...model.SmartServiceRelease) {
		t.Helper()
		result, code := getUsage(t, token, kind, id)
		if code != http.StatusOK {
			t.Errorf("%v %v: status %v", kind, id, code)
			return
		}
		expected := model.ResourceUsage{Releases: releases, Instances: instances, Readable: []model.ResourceUsageRelease{}}
		for _, r := range readable {
			expected.Readable = append(expected.Readable, model.ResourceUsageRelease{Id: r.Id, DesignId: r.DesignId, Name: r.Name})
		}
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("%v %v:\nexpected %#v\ngot      %#v", kind, id, expected, result)
		}
	}

	expectStatus := func(t *testing.T, token string, path string, expected int) {
		t.Helper()
		resp, err := get(token, apiUrl+path)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != expected {
			temp, _ := io.ReadAll(resp.Body)
			t.Error(path, resp.StatusCode, string(temp))
		}
	}

	designA := createDesign(t, usageBpmn("pm-a", "flow-a", "it-a"))
	a1 := createRelease(t, designA, "a1")
	a2 := createRelease(t, designA, "a2")
	a3 := createRelease(t, designA, "a3")

	t.Run("release stores the literal ids it references", func(t *testing.T) {
		stored, err, _ := db.GetRelease(ctx, a3.Id, false)
		if err != nil {
			t.Fatal(err)
		}
		expected := &model.ReleaseUsedResources{ProcessModels: []string{"pm-a"}, Flows: []string{"flow-a"}, ImportTypes: []string{"it-a"}}
		if !reflect.DeepEqual(stored.UsedResources, expected) {
			t.Errorf("\nexpected %#v\ngot      %#v", expected, stored.UsedResources)
		}
	})

	t.Run("only the newest release counts while there are no instances", func(t *testing.T) {
		expectUsage(t, userToken, "process-models", "pm-a", 1, 0, a3)
		expectUsage(t, userToken, "flows", "flow-a", 1, 0, a3)
		expectUsage(t, userToken, "import-types", "it-a", 1, 0, a3)
	})

	t.Run("an older release counts once it has instances, of any user", func(t *testing.T) {
		createInstance(t, userId, a1)
		createInstance(t, otherUserId, a1)
		expectUsage(t, userToken, "process-models", "pm-a", 2, 2, a1, a3)
		expectUsage(t, userToken, "flows", "flow-a", 2, 2, a1, a3)
		expectUsage(t, userToken, "import-types", "it-a", 2, 2, a1, a3)
		_ = a2 // older and without instances: never counted
	})

	t.Run("unused and expression ids are not counted", func(t *testing.T) {
		expectUsage(t, userToken, "process-models", "pm-unused", 0, 0)
		expectUsage(t, userToken, "process-models", "${model_id}", 0, 0)
		expectUsage(t, userToken, "flows", "pm-a", 0, 0)
	})

	t.Run("deleted release is not counted", func(t *testing.T) {
		designB := createDesign(t, usageBpmn("pm-a", "flow-b", "it-b"))
		b1 := createRelease(t, designB, "b1")
		expectUsage(t, userToken, "process-models", "pm-a", 3, 2, a1, a3, b1)
		// the state while deleteRelease() runs, before the document is removed
		err, _ := db.MarlReleaseAsDeleted(ctx, b1.Id)
		if err != nil {
			t.Fatal(err)
		}
		expectUsage(t, userToken, "process-models", "pm-a", 2, 2, a1, a3)
		resp, err := delete(userToken, apiUrl+"/releases/"+url.PathEscape(b1.Id))
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatal(resp.StatusCode)
		}
		expectUsage(t, userToken, "process-models", "pm-a", 2, 2, a1, a3)
		expectUsage(t, userToken, "flows", "flow-b", 0, 0)
	})

	t.Run("release whose successor is marked as deleted counts as newest", func(t *testing.T) {
		designC := createDesign(t, usageBpmn("pm-c", "flow-c", "it-c"))
		c1 := createRelease(t, designC, "c1")
		c2 := createRelease(t, designC, "c2")
		expectUsage(t, userToken, "process-models", "pm-c", 1, 0, c2)
		err, _ := db.MarlReleaseAsDeleted(ctx, c2.Id)
		if err != nil {
			t.Fatal(err)
		}
		expectUsage(t, userToken, "process-models", "pm-c", 1, 0, c1)
	})

	t.Run("release whose successor is still unfinished counts as newest", func(t *testing.T) {
		designE := createDesign(t, usageBpmn("pm-e", "flow-e", "it-e"))
		e1 := createRelease(t, designE, "e1")
		e2 := createRelease(t, designE, "e2")
		expectUsage(t, userToken, "process-models", "pm-e", 1, 0, e2)
		// the state after a creation whose MarkReleaseAsFinished() failed: the cleanup will delete e2 and make e1 the newest again
		stored, err, _ := db.GetRelease(ctx, e2.Id, true)
		if err != nil {
			t.Fatal(err)
		}
		err, _ = db.SetRelease(ctx, stored, true)
		if err != nil {
			t.Fatal(err)
		}
		expectUsage(t, userToken, "process-models", "pm-e", 2, 0, e1, e2)
	})

	t.Run("readable lists only the releases the caller may read, the counts cover all users", func(t *testing.T) {
		expectUsage(t, otherUserToken(), "process-models", "pm-a", 2, 2)
		_, err, _ := perm.SetPermission(adminToken, config.SmartServiceReleasePermissionsTopic, a3.Id, permclient.ResourcePermissions{
			UserPermissions: map[string]permmodel.PermissionsMap{
				userId:      {Read: true, Write: true, Execute: true, Administrate: true},
				otherUserId: {Read: true},
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		expectUsage(t, otherUserToken(), "process-models", "pm-a", 2, 2, a3)
		expectUsage(t, userToken, "process-models", "pm-a", 2, 2, a1, a3)
	})

	t.Run("a new release keeps used resources on the older release it updates", func(t *testing.T) {
		designD := createDesign(t, usageBpmn("pm-d", "flow-d", "it-d"))
		storeLegacyRelease(t, "legacy-d", designD.Id, usageBpmn("pm-d-old", "flow-d-old", "it-d-old"))
		d2 := createRelease(t, designD, "d2")
		stored, err, _ := db.GetRelease(ctx, "legacy-d", true)
		if err != nil {
			t.Fatal(err)
		}
		if stored.NewReleaseId != d2.Id {
			t.Fatalf("expected new_release_id %v, got %v", d2.Id, stored.NewReleaseId)
		}
		expected := &model.ReleaseUsedResources{ProcessModels: []string{"pm-d-old"}, Flows: []string{"flow-d-old"}, ImportTypes: []string{"it-d-old"}}
		if !reflect.DeepEqual(stored.UsedResources, expected) {
			t.Errorf("\nexpected %#v\ngot      %#v", expected, stored.UsedResources)
		}
	})

	t.Run("query fills a release stored without used resources", func(t *testing.T) {
		storeLegacyRelease(t, "legacy-1", "design-legacy-1", usageBpmn("pm-legacy", "flow-legacy", "it-legacy"))
		// the legacy release has no permissions entry, so it counts but nobody may read it
		expectUsage(t, userToken, "process-models", "pm-legacy", 1, 0)
		stored, err, _ := db.GetRelease(ctx, "legacy-1", true)
		if err != nil {
			t.Fatal(err)
		}
		expected := &model.ReleaseUsedResources{ProcessModels: []string{"pm-legacy"}, Flows: []string{"flow-legacy"}, ImportTypes: []string{"it-legacy"}}
		if !reflect.DeepEqual(stored.UsedResources, expected) {
			t.Errorf("\nexpected %#v\ngot      %#v", expected, stored.UsedResources)
		}
	})

	t.Run("backfill fills releases stored without used resources once", func(t *testing.T) {
		storeLegacyRelease(t, "legacy-2", "design-legacy-2", usageBpmn("pm-legacy-2", "flow-legacy-2", "it-legacy-2"))
		ctrl, err := controller.New(ctx, config, db, perm, camunda.New(config), nil, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		filled, unparsable, err := ctrl.BackfillReleaseUsedResources(ctx)
		if err != nil || filled != 1 || unparsable != 0 {
			t.Fatalf("first backfill: filled=%v unparsable=%v err=%v", filled, unparsable, err)
		}
		stored, err, _ := db.GetRelease(ctx, "legacy-2", true)
		if err != nil {
			t.Fatal(err)
		}
		expected := &model.ReleaseUsedResources{ProcessModels: []string{"pm-legacy-2"}, Flows: []string{"flow-legacy-2"}, ImportTypes: []string{"it-legacy-2"}}
		if !reflect.DeepEqual(stored.UsedResources, expected) {
			t.Errorf("\nexpected %#v\ngot      %#v", expected, stored.UsedResources)
		}
		filled, unparsable, err = ctrl.BackfillReleaseUsedResources(ctx)
		if err != nil || filled != 0 || unparsable != 0 {
			t.Fatalf("second backfill: filled=%v unparsable=%v err=%v", filled, unparsable, err)
		}
	})

	t.Run("a release with unparsable bpmn is indexed as using nothing and does not fail the query", func(t *testing.T) {
		storeLegacyRelease(t, "legacy-broken", "design-legacy-broken", "<bpmn:definitions")
		expectUsage(t, userToken, "process-models", "pm-a", 2, 2, a1, a3)
		stored, err, _ := db.GetRelease(ctx, "legacy-broken", true)
		if err != nil {
			t.Fatal(err)
		}
		expected := &model.ReleaseUsedResources{ProcessModels: []string{}, Flows: []string{}, ImportTypes: []string{}, Unparsable: true}
		if !reflect.DeepEqual(stored.UsedResources, expected) {
			t.Errorf("\nexpected %#v\ngot      %#v", expected, stored.UsedResources)
		}
		ctrl, err := controller.New(ctx, config, db, perm, camunda.New(config), nil, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		filled, unparsable, err := ctrl.BackfillReleaseUsedResources(ctx)
		if err != nil || filled != 0 || unparsable != 0 {
			t.Errorf("backfill: filled=%v unparsable=%v err=%v", filled, unparsable, err)
		}
	})

	t.Run("400 for an unknown kind", func(t *testing.T) {
		expectStatus(t, userToken, "/resource-usage/designs/pm-a", http.StatusBadRequest)
		expectStatus(t, userToken, "/resource-usage/process-model/pm-a", http.StatusBadRequest)
	})

	t.Run("400 for an empty id", func(t *testing.T) {
		expectStatus(t, userToken, "/resource-usage/flows/%20", http.StatusBadRequest)
	})

	t.Run("401 without token", func(t *testing.T) {
		expectStatus(t, "", "/resource-usage/process-models/pm-a", http.StatusUnauthorized)
	})
}
