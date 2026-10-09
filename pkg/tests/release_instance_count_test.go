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
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"runtime/debug"
	"sync"
	"testing"

	permclient "github.com/SENERGY-Platform/permissions-v2/pkg/client"
	permmodel "github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/database/mongo"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/tests/resources"
	"github.com/google/uuid"
)

const otherUserId = "otherUserId"

// otherUserToken is a non-admin token of a user without any permissions on the test releases.
// The signature is not checked by the service, like for the other test tokens.
func otherUserToken() string {
	enc := base64.RawURLEncoding.EncodeToString
	header, _ := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	claims, _ := json.Marshal(map[string]interface{}{
		"sub":          otherUserId,
		"realm_access": map[string]interface{}{"roles": []string{"user"}},
	})
	return "Bearer " + enc(header) + "." + enc(claims) + "." + enc([]byte("signature"))
}

func TestReleaseInstanceCount(t *testing.T) {
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

	// instances are written directly: starting them over the api needs a camunda that answers like the real one
	db, err := mongo.New(config)
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
		err = json.NewDecoder(resp.Body).Decode(&design)
		if err != nil {
			t.Error(err)
			return
		}
	})

	createRelease := func(t *testing.T, name string) (release model.SmartServiceRelease) {
		t.Helper()
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
		err = json.NewDecoder(resp.Body).Decode(&release)
		if err != nil {
			t.Error(err)
		}
		return
	}

	createInstance := func(t *testing.T, instanceUserId string, release model.SmartServiceRelease, name string) {
		t.Helper()
		err, _ := db.SetInstance(ctx, model.SmartServiceInstance{
			SmartServiceInstanceInit: model.SmartServiceInstanceInit{
				SmartServiceInstanceInfo: model.SmartServiceInstanceInfo{Name: name},
			},
			Id:        uuid.NewString(),
			UserId:    instanceUserId,
			DesignId:  release.DesignId,
			ReleaseId: release.Id,
		})
		if err != nil {
			t.Error(err)
		}
	}

	// expects the exact body {"count": n}: no further keys, so no ids or user ids can leak
	expectCount := func(t *testing.T, token string, releaseId string, expected int) {
		t.Helper()
		resp, err := get(token, apiUrl+"/releases/"+url.PathEscape(releaseId)+"/instance-count")
		if err != nil {
			t.Error(err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			temp, _ := io.ReadAll(resp.Body)
			t.Error(resp.StatusCode, string(temp))
			return
		}
		checkContentType(t, resp)
		result := map[string]interface{}{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		if err != nil {
			t.Error(err)
			return
		}
		if len(result) != 1 || result["count"] != float64(expected) {
			t.Errorf("expected {\"count\": %v}, got %v", expected, result)
		}
	}

	expectStatus := func(t *testing.T, token string, releaseId string, expected int) {
		t.Helper()
		resp, err := get(token, apiUrl+"/releases/"+url.PathEscape(releaseId)+"/instance-count")
		if err != nil {
			t.Error(err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != expected {
			temp, _ := io.ReadAll(resp.Body)
			t.Error(resp.StatusCode, string(temp))
		}
	}

	release1 := createRelease(t, "release 1")
	release2 := createRelease(t, "release 2")

	t.Run("zero instances", func(t *testing.T) {
		expectCount(t, userToken, release1.Id, 0)
	})

	t.Run("one instance", func(t *testing.T) {
		createInstance(t, userId, release1, "instance 1")
		expectCount(t, userToken, release1.Id, 1)
	})

	t.Run("several instances of different users", func(t *testing.T) {
		createInstance(t, userId, release1, "instance 2")
		createInstance(t, adminId, release1, "instance 3")
		createInstance(t, otherUserId, release1, "instance 4")
		expectCount(t, userToken, release1.Id, 4)
	})

	t.Run("counts only the given release", func(t *testing.T) {
		expectCount(t, userToken, release2.Id, 0)
		createInstance(t, userId, release2, "instance 5")
		expectCount(t, userToken, release2.Id, 1)
		expectCount(t, userToken, release1.Id, 4)
	})

	t.Run("403 without administrate", func(t *testing.T) {
		expectStatus(t, otherUserToken(), release1.Id, http.StatusForbidden)
	})

	t.Run("403 with read and write but without administrate", func(t *testing.T) {
		_, err, _ := perm.SetPermission(adminToken, config.SmartServiceReleasePermissionsTopic, release1.Id, permclient.ResourcePermissions{
			UserPermissions: map[string]permmodel.PermissionsMap{
				userId:      {Read: true, Write: true, Execute: true, Administrate: true},
				otherUserId: {Read: true, Write: true, Execute: true},
			},
		})
		if err != nil {
			t.Error(err)
			return
		}
		expectStatus(t, otherUserToken(), release1.Id, http.StatusForbidden)
		expectCount(t, userToken, release1.Id, 4)
	})

	t.Run("401 without token", func(t *testing.T) {
		expectStatus(t, "", release1.Id, http.StatusUnauthorized)
	})

	// permissions answer 403 for ids they do not know, so only an admin reaches the release lookup
	t.Run("404 for unknown release", func(t *testing.T) {
		expectStatus(t, adminToken, "unknown-release-id", http.StatusNotFound)
	})

	t.Run("404 for deleted release", func(t *testing.T) {
		release3 := createRelease(t, "release 3")
		expectCount(t, userToken, release3.Id, 0)
		resp, err := delete(userToken, apiUrl+"/releases/"+url.PathEscape(release3.Id))
		if err != nil {
			t.Error(err)
			return
		}
		if resp.StatusCode != http.StatusOK {
			temp, _ := io.ReadAll(resp.Body)
			t.Error(resp.StatusCode, string(temp))
			return
		}
		expectStatus(t, adminToken, release3.Id, http.StatusNotFound)
	})
}
