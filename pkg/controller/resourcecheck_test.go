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

package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/SENERGY-Platform/smart-service-repository/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
)

// resourceCheckTestDb implements only the methods the resource check uses; every other Database method panics
type resourceCheckTestDb struct {
	Database
	mux      sync.Mutex
	releases []model.SmartServiceReleaseExtended
	writes   int
}

func (this *resourceCheckTestDb) ListReleasesWithUsedResources(ctx context.Context) ([]model.SmartServiceReleaseExtended, error) {
	this.mux.Lock()
	defer this.mux.Unlock()
	return slices.Clone(this.releases), nil
}

func (this *resourceCheckTestDb) SetReleaseResourceCheck(ctx context.Context, id string, changed []model.ChangedResource, hashes map[string]string) error {
	this.mux.Lock()
	defer this.mux.Unlock()
	this.writes++
	for i := range this.releases {
		if this.releases[i].Id == id {
			this.releases[i].ChangedResources = changed
			this.releases[i].ImportTypeHashes = hashes
			if len(changed) == 0 {
				this.releases[i].ChangedResources = nil
			}
		}
	}
	return nil
}

func (this *resourceCheckTestDb) release(id string) model.SmartServiceReleaseExtended {
	this.mux.Lock()
	defer this.mux.Unlock()
	for _, release := range this.releases {
		if release.Id == id {
			return release
		}
	}
	panic("unknown release " + id)
}

const fakeAdminToken = "Bearer admin-token"

// fakeRepo answers GET {path} like the repositories: a known id answers from the maps, an unknown id answers 403 to a user
// token (the permission is checked before the existence) and 404 to the token of the service.
// A response set for the user is also the answer to the service unless one is set for the service.
type fakeRepo struct {
	*httptest.Server
	mux           sync.Mutex
	responses     map[string]fakeResponse
	adminResponse map[string]fakeResponse
	hang          map[string]bool
	requests      int
	adminRequests int
	auth          []string
}

type fakeResponse struct {
	status int
	body   string
}

func newFakeRepo(t *testing.T) *fakeRepo {
	repo := &fakeRepo{responses: map[string]fakeResponse{}, adminResponse: map[string]fakeResponse{}, hang: map[string]bool{}}
	repo.Server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		repo.mux.Lock()
		repo.requests++
		authorization := request.Header.Get("Authorization")
		repo.auth = append(repo.auth, authorization)
		isAdmin := authorization == fakeAdminToken
		if isAdmin {
			repo.adminRequests++
		}
		hang := repo.hang[request.URL.Path]
		response, known := repo.responses[request.URL.Path]
		if adminResponse, ok := repo.adminResponse[request.URL.Path]; isAdmin && ok {
			response, known = adminResponse, true
		}
		repo.mux.Unlock()
		if hang {
			<-request.Context().Done()
			return
		}
		if !known {
			response = fakeResponse{status: http.StatusForbidden}
			if isAdmin {
				response = fakeResponse{status: http.StatusNotFound}
			}
		}
		writer.WriteHeader(response.status)
		_, _ = writer.Write([]byte(response.body))
	}))
	t.Cleanup(repo.Close)
	return repo
}

func (this *fakeRepo) set(path string, status int, body string) {
	this.mux.Lock()
	defer this.mux.Unlock()
	this.responses[path] = fakeResponse{status: status, body: body}
}

func (this *fakeRepo) setForAdmin(path string, status int, body string) {
	this.mux.Lock()
	defer this.mux.Unlock()
	this.adminResponse[path] = fakeResponse{status: status, body: body}
}

func (this *fakeRepo) setHang(path string) {
	this.mux.Lock()
	defer this.mux.Unlock()
	this.hang[path] = true
}

func (this *fakeRepo) requestCount() int {
	this.mux.Lock()
	defer this.mux.Unlock()
	return this.requests
}

func (this *fakeRepo) adminRequestCount() int {
	this.mux.Lock()
	defer this.mux.Unlock()
	return this.adminRequests
}

type resourceCheckFixture struct {
	t    *testing.T
	db   *resourceCheckTestDb
	ctrl *Controller
	pm   *fakeRepo
	flow *fakeRepo
	imp  *fakeRepo
	now  int64

	keycloakStatus int //0 answers with the admin token
}

const checkTestCreatedAt = int64(1000)

func newResourceCheckFixture(t *testing.T, used model.ReleaseUsedResources, previous ...model.ChangedResource) *resourceCheckFixture {
	f := &resourceCheckFixture{t: t, now: 5000}
	f.pm = newFakeRepo(t)
	f.flow = newFakeRepo(t)
	f.imp = newFakeRepo(t)
	f.db = &resourceCheckTestDb{releases: []model.SmartServiceReleaseExtended{{
		SmartServiceRelease: model.SmartServiceRelease{Id: "r1", CreatedAt: checkTestCreatedAt, Creator: "creator-1", ChangedResources: previous},
		UsedResources:       &used,
	}}}
	keycloak := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if f.keycloakStatus != 0 {
			writer.WriteHeader(f.keycloakStatus)
			return
		}
		_ = json.NewEncoder(writer).Encode(auth.OpenidToken{AccessToken: strings.TrimPrefix(fakeAdminToken, "Bearer "), ExpiresIn: 60})
	}))
	t.Cleanup(keycloak.Close)
	f.ctrl = &Controller{
		adminAccess: &auth.OpenidToken{},
		config: configuration.Config{
			AuthEndpoint:        keycloak.URL,
			ProcessModelRepoUrl: f.pm.URL,
			FlowRepoUrl:         f.flow.URL + "/", // a trailing slash must not matter
			ImportRepoUrl:       f.imp.URL,
		},
		db: f.db,
		userTokenProvider: func(userid string) (auth.Token, error) {
			return auth.Token{Token: "Bearer token-of-" + userid}, nil
		},
		clock: func() time.Time { return time.Unix(f.now, 0) },
	}
	return f
}

func (this *resourceCheckFixture) run() ResourceCheckResult {
	this.t.Helper()
	result, err := this.ctrl.CheckReleaseResources(context.Background())
	if err != nil {
		this.t.Fatal(err)
	}
	return result
}

func (this *resourceCheckFixture) expectChanged(expected ...model.ChangedResource) {
	this.t.Helper()
	got := this.db.release("r1").ChangedResources
	if len(expected) == 0 && len(got) == 0 {
		return
	}
	if !reflect.DeepEqual(got, expected) {
		this.t.Errorf("\nexpected %#v\ngot      %#v", expected, got)
	}
}

func (this *resourceCheckFixture) expectWrites(expected int) {
	this.t.Helper()
	if this.db.writes != expected {
		this.t.Errorf("expected %v writes, got %v", expected, this.db.writes)
	}
}

func processModelJson(updated int64) string {
	temp, _ := json.Marshal(map[string]interface{}{"id": "pm1", "last_updated_unix": updated})
	return string(temp)
}

func TestCheckReleaseResources(t *testing.T) {
	pmUsed := model.ReleaseUsedResources{ProcessModels: []string{"pm1"}}
	flowUsed := model.ReleaseUsedResources{Flows: []string{"flow1"}}
	importUsed := model.ReleaseUsedResources{ImportTypes: []string{"it1"}}

	t.Run("process model changed after the release was created", func(t *testing.T) {
		f := newResourceCheckFixture(t, pmUsed)
		f.pm.set("/processes/pm1", 200, processModelJson(checkTestCreatedAt+1))
		result := f.run()
		if result != (ResourceCheckResult{Checked: 1, Marked: 1}) {
			t.Errorf("%#v", result)
		}
		f.expectChanged(model.ChangedResource{Kind: "process_model", Id: "pm1", Reason: "changed", At: checkTestCreatedAt + 1})
		f.expectWrites(1)
		if !slices.Equal(f.pm.auth, []string{"Bearer token-of-creator-1"}) {
			t.Errorf("expected the token of the release creator, got %v", f.pm.auth)
		}
	})

	t.Run("process model changed before or at the creation of the release", func(t *testing.T) {
		for _, updated := range []int64{checkTestCreatedAt - 1, checkTestCreatedAt, 0} {
			f := newResourceCheckFixture(t, pmUsed)
			f.pm.set("/processes/pm1", 200, processModelJson(updated))
			result := f.run()
			if result != (ResourceCheckResult{Checked: 1}) {
				t.Errorf("updated %v: %#v", updated, result)
			}
			f.expectChanged()
			f.expectWrites(0)
		}
	})

	t.Run("flow changed after the release was created", func(t *testing.T) {
		f := newResourceCheckFixture(t, flowUsed)
		updated := time.Unix(checkTestCreatedAt+60, 0).UTC().Format(time.RFC3339)
		f.flow.set("/flow/flow1", 200, `{"id":"flow1","dateUpdated":"`+updated+`"}`)
		f.run()
		f.expectChanged(model.ChangedResource{Kind: "flow", Id: "flow1", Reason: "changed", At: checkTestCreatedAt + 60})
	})

	t.Run("flow without a change time is not reported", func(t *testing.T) {
		for _, body := range []string{`{"id":"flow1"}`, `{"id":"flow1","dateUpdated":"0001-01-01T00:00:00Z"}`, `{"id":"flow1","dateUpdated":""}`} {
			f := newResourceCheckFixture(t, flowUsed)
			f.flow.set("/flow/flow1", 200, body)
			result := f.run()
			if result.Errors != 0 {
				t.Errorf("%v: %#v", body, result)
			}
			f.expectChanged()
			f.expectWrites(0)
		}
	})

	t.Run("import type: first run stores the baseline, a later change marks it and keeps the first time", func(t *testing.T) {
		f := newResourceCheckFixture(t, importUsed)
		f.imp.set("/import-types/it1", 200, `{"name":"a","version":1}`)
		f.run()
		f.expectChanged()
		f.expectWrites(1)
		baseline := f.db.release("r1").ImportTypeHashes["it1"]
		if baseline == "" {
			t.Fatal("no baseline stored")
		}

		// the same content with another key order and spacing hashes the same
		f.imp.set("/import-types/it1", 200, `{ "version": 1, "name": "a" }`)
		f.run()
		f.expectChanged()
		f.expectWrites(1)

		f.imp.set("/import-types/it1", 200, `{"name":"b","version":1}`)
		f.now = 6000
		result := f.run()
		if result != (ResourceCheckResult{Checked: 1, Marked: 1}) {
			t.Errorf("%#v", result)
		}
		f.expectChanged(model.ChangedResource{Kind: "import_type", Id: "it1", Reason: "changed", At: 6000})
		f.expectWrites(2)
		if f.db.release("r1").ImportTypeHashes["it1"] != baseline {
			t.Error("the baseline must not move")
		}

		f.now = 7000
		f.run()
		f.expectChanged(model.ChangedResource{Kind: "import_type", Id: "it1", Reason: "changed", At: 6000})
		f.expectWrites(2)

		f.imp.set("/import-types/it1", 200, `{"name":"a","version":1}`)
		f.run()
		f.expectChanged()
		f.expectWrites(3)
	})

	t.Run("404 marks the resource as gone and keeps the first time", func(t *testing.T) {
		f := newResourceCheckFixture(t, model.ReleaseUsedResources{ProcessModels: []string{"pm1"}, Flows: []string{"flow1"}, ImportTypes: []string{"it1"}})
		result := f.run()
		if result != (ResourceCheckResult{Checked: 1, Marked: 1}) {
			t.Errorf("%#v", result)
		}
		expected := []model.ChangedResource{
			{Kind: "process_model", Id: "pm1", Reason: "gone", At: 5000},
			{Kind: "flow", Id: "flow1", Reason: "gone", At: 5000},
			{Kind: "import_type", Id: "it1", Reason: "gone", At: 5000},
		}
		f.expectChanged(expected...)
		f.expectWrites(1)

		f.now = 9000
		f.run()
		f.expectChanged(expected...)
		f.expectWrites(1)
	})

	t.Run("a gone resource that is back and unchanged drops out", func(t *testing.T) {
		f := newResourceCheckFixture(t, pmUsed, model.ChangedResource{Kind: "process_model", Id: "pm1", Reason: "gone", At: 4000})
		f.pm.set("/processes/pm1", 200, processModelJson(checkTestCreatedAt))
		f.run()
		f.expectChanged()
		f.expectWrites(1)
	})

	t.Run("an unreadable resource keeps its previous entry", func(t *testing.T) {
		previous := model.ChangedResource{Kind: "process_model", Id: "pm1", Reason: "changed", At: 4000}
		for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusInternalServerError} {
			f := newResourceCheckFixture(t, pmUsed, previous)
			f.pm.set("/processes/pm1", status, "")
			result := f.run()
			if result != (ResourceCheckResult{Checked: 1, Marked: 1, Errors: 1}) {
				t.Errorf("status %v: %#v", status, result)
			}
			f.expectChanged(previous)
			f.expectWrites(0)
		}
	})

	t.Run("an unreadable resource without previous entry stays unmarked", func(t *testing.T) {
		f := newResourceCheckFixture(t, pmUsed)
		f.pm.set("/processes/pm1", http.StatusForbidden, "")
		result := f.run()
		if result != (ResourceCheckResult{Checked: 1, Errors: 1}) {
			t.Errorf("%#v", result)
		}
		f.expectChanged()
		f.expectWrites(0)
	})

	t.Run("an unknown id answers 403 to the creator and 404 to the service: gone", func(t *testing.T) {
		f := newResourceCheckFixture(t, pmUsed)
		result := f.run()
		if result != (ResourceCheckResult{Checked: 1, Marked: 1}) {
			t.Errorf("%#v", result)
		}
		f.expectChanged(model.ChangedResource{Kind: "process_model", Id: "pm1", Reason: "gone", At: 5000})
		if !slices.Equal(f.pm.auth, []string{"Bearer token-of-creator-1", fakeAdminToken}) {
			t.Errorf("expected the creator first, then the service, got %v", f.pm.auth)
		}
	})

	t.Run("a resource the creator may not read is compared with the answer of the service", func(t *testing.T) {
		f := newResourceCheckFixture(t, pmUsed)
		f.pm.set("/processes/pm1", http.StatusForbidden, "")
		f.pm.setForAdmin("/processes/pm1", http.StatusOK, processModelJson(checkTestCreatedAt+1))
		f.run()
		f.expectChanged(model.ChangedResource{Kind: "process_model", Id: "pm1", Reason: "changed", At: checkTestCreatedAt + 1})

		// unchanged: a gone entry drops out although the creator still cannot read it
		f = newResourceCheckFixture(t, pmUsed, model.ChangedResource{Kind: "process_model", Id: "pm1", Reason: "gone", At: 4000})
		f.pm.set("/processes/pm1", http.StatusUnauthorized, "")
		f.pm.setForAdmin("/processes/pm1", http.StatusOK, processModelJson(checkTestCreatedAt))
		result := f.run()
		if result != (ResourceCheckResult{Checked: 1}) {
			t.Errorf("%#v", result)
		}
		f.expectChanged()
	})

	t.Run("the service is asked only after 401 or 403", func(t *testing.T) {
		f := newResourceCheckFixture(t, pmUsed)
		f.pm.set("/processes/pm1", http.StatusInternalServerError, "")
		result := f.run()
		if result != (ResourceCheckResult{Checked: 1, Errors: 1}) || f.pm.adminRequestCount() != 0 {
			t.Errorf("%#v, %v admin requests", result, f.pm.adminRequestCount())
		}
		f = newResourceCheckFixture(t, pmUsed)
		f.pm.set("/processes/pm1", http.StatusOK, processModelJson(checkTestCreatedAt+1))
		f.run()
		if f.pm.adminRequestCount() != 0 {
			t.Errorf("%v admin requests", f.pm.adminRequestCount())
		}
	})

	t.Run("a failing answer of the service keeps the previous entry", func(t *testing.T) {
		previous := model.ChangedResource{Kind: "process_model", Id: "pm1", Reason: "changed", At: 4000}
		for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusInternalServerError} {
			f := newResourceCheckFixture(t, pmUsed, previous)
			f.pm.set("/processes/pm1", http.StatusForbidden, "")
			f.pm.setForAdmin("/processes/pm1", status, "")
			result := f.run()
			if result != (ResourceCheckResult{Checked: 1, Marked: 1, Errors: 1}) {
				t.Errorf("service status %v: %#v", status, result)
			}
			f.expectChanged(previous)
			f.expectWrites(0)
		}
	})

	t.Run("without a token of the service the previous entry stays", func(t *testing.T) {
		previous := model.ChangedResource{Kind: "process_model", Id: "pm1", Reason: "changed", At: 4000}
		f := newResourceCheckFixture(t, pmUsed, previous)
		f.keycloakStatus = http.StatusInternalServerError
		result := f.run()
		if result != (ResourceCheckResult{Checked: 1, Marked: 1, Errors: 1}) {
			t.Errorf("%#v", result)
		}
		f.expectChanged(previous)
		f.expectWrites(0)
	})

	t.Run("a hanging repository costs only its release", func(t *testing.T) {
		f := newResourceCheckFixture(t, model.ReleaseUsedResources{ProcessModels: []string{"pm-hang"}})
		f.ctrl.releaseCheckTimeout = 200 * time.Millisecond
		f.pm.setHang("/processes/pm-hang")
		f.pm.set("/processes/pm2", http.StatusOK, processModelJson(checkTestCreatedAt+1))
		second := model.SmartServiceReleaseExtended{
			SmartServiceRelease: model.SmartServiceRelease{Id: "r2", CreatedAt: checkTestCreatedAt, Creator: "creator-1"},
			UsedResources:       &model.ReleaseUsedResources{ProcessModels: []string{"pm2"}},
		}
		f.db.releases = append(f.db.releases, second)
		start := time.Now()
		result := f.run()
		if time.Since(start) > 5*time.Second {
			t.Errorf("the run waited %v for the hanging repository", time.Since(start))
		}
		if result != (ResourceCheckResult{Checked: 2, Marked: 1, Errors: 1}) {
			t.Errorf("%#v", result)
		}
		if got := f.db.release("r2").ChangedResources; len(got) != 1 || got[0].Id != "pm2" {
			t.Errorf("%#v", got)
		}
	})

	t.Run("a resource that is readable and unchanged does not cause a write", func(t *testing.T) {
		f := newResourceCheckFixture(t, model.ReleaseUsedResources{ProcessModels: []string{"pm1"}, Flows: []string{"flow1"}})
		f.pm.set("/processes/pm1", 200, processModelJson(checkTestCreatedAt-5))
		f.flow.set("/flow/flow1", 200, `{"dateUpdated":"`+time.Unix(checkTestCreatedAt-5, 0).UTC().Format(time.RFC3339)+`"}`)
		f.run()
		f.run()
		f.expectChanged()
		f.expectWrites(0)
	})

	t.Run("a kind without url is skipped and its entries stay", func(t *testing.T) {
		previous := model.ChangedResource{Kind: "flow", Id: "flow1", Reason: "changed", At: 4000}
		f := newResourceCheckFixture(t, model.ReleaseUsedResources{ProcessModels: []string{"pm1"}, Flows: []string{"flow1"}}, previous)
		f.ctrl.config.FlowRepoUrl = ""
		f.pm.set("/processes/pm1", 200, processModelJson(checkTestCreatedAt+1))
		f.run()
		f.expectChanged(
			model.ChangedResource{Kind: "process_model", Id: "pm1", Reason: "changed", At: checkTestCreatedAt + 1},
			previous,
		)
		if f.flow.requestCount() != 0 {
			t.Errorf("flow repository was asked %v times", f.flow.requestCount())
		}
	})

	t.Run("without any url nothing is requested and no token is fetched", func(t *testing.T) {
		f := newResourceCheckFixture(t, model.ReleaseUsedResources{ProcessModels: []string{"pm1"}})
		f.ctrl.config.ProcessModelRepoUrl = ""
		f.ctrl.userTokenProvider = func(userid string) (auth.Token, error) {
			t.Error("token requested")
			return auth.Token{}, nil
		}
		f.run()
		f.expectWrites(0)
	})

	t.Run("a release without used resources is skipped", func(t *testing.T) {
		f := newResourceCheckFixture(t, pmUsed)
		f.db.releases[0].UsedResources = nil
		result := f.run()
		if result != (ResourceCheckResult{Checked: 1}) || f.pm.requestCount() != 0 {
			t.Errorf("%#v, %v requests", result, f.pm.requestCount())
		}
	})

	t.Run("a failing token provider is counted and leaves the release as it is", func(t *testing.T) {
		previous := model.ChangedResource{Kind: "process_model", Id: "pm1", Reason: "changed", At: 4000}
		f := newResourceCheckFixture(t, pmUsed, previous)
		f.ctrl.userTokenProvider = func(userid string) (auth.Token, error) {
			return auth.Token{}, http.ErrNoCookie
		}
		result := f.run()
		if result != (ResourceCheckResult{Checked: 1, Marked: 1, Errors: 1}) {
			t.Errorf("%#v", result)
		}
		f.expectChanged(previous)
		f.expectWrites(0)
	})

	t.Run("runs never overlap", func(t *testing.T) {
		f := newResourceCheckFixture(t, pmUsed)
		slow := &overlapTestDb{resourceCheckTestDb: f.db}
		f.ctrl.db = slow
		wg := sync.WaitGroup{}
		for i := 0; i < 4; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _ = f.ctrl.CheckReleaseResources(context.Background())
			}()
		}
		wg.Wait()
		if slow.maxActive != 1 {
			t.Errorf("expected one run at a time, got %v", slow.maxActive)
		}
	})
}

type overlapTestDb struct {
	*resourceCheckTestDb
	active    int
	maxActive int
}

func (this *overlapTestDb) ListReleasesWithUsedResources(ctx context.Context) ([]model.SmartServiceReleaseExtended, error) {
	this.mux.Lock()
	this.active++
	this.maxActive = max(this.maxActive, this.active)
	this.mux.Unlock()
	time.Sleep(20 * time.Millisecond)
	this.mux.Lock()
	this.active--
	this.mux.Unlock()
	return this.resourceCheckTestDb.ListReleasesWithUsedResources(ctx)
}

// the baseline of the import types is internal and must not be part of any json answer
func TestImportTypeHashesAreNotServed(t *testing.T) {
	release := model.SmartServiceReleaseExtended{ImportTypeHashes: map[string]string{"it1": "secret-hash"}}
	temp, err := json.Marshal(release)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(temp), "secret-hash") || strings.Contains(string(temp), "it1") {
		t.Errorf("import type hashes in %v", string(temp))
	}
}

func TestImportTypeHash(t *testing.T) {
	base := `{
		"id": "it1", "name": "a", "description": "d", "image": "img", "default_restart": true,
		"configs": [{"name": "c", "description": "cd", "type": "https://schema.org/Text", "default_value": {"b": 1, "a": [1, 2]}}],
		"output": {"name": "root", "type": "https://schema.org/StructuredValue", "characteristic_id": "", "use_as_tag": false,
			"sub_content_variables": [{"name": "t", "type": "https://schema.org/Float", "characteristic_id": "char", "use_as_tag": true, "function_id": "fn", "aspect_ids": ["asp"]}]}
	}`
	hash := func(body string) string {
		t.Helper()
		result, err := importTypeHash([]byte(body))
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	expected := hash(base)

	same := map[string]string{
		"extra field, other owner and cost, other id": `{
			"id": "other", "owner": "someone", "cost": 5, "new_field": {"x": 1}, "name": "a", "description": "d", "image": "img", "default_restart": true,
			"configs": [{"name": "c", "description": "cd", "type": "https://schema.org/Text", "default_value": {"a": [1, 2], "b": 1}, "extra": 1}],
			"output": {"name": "root", "type": "https://schema.org/StructuredValue", "characteristic_id": "", "use_as_tag": false, "extra": 1,
				"sub_content_variables": [{"name": "t", "type": "https://schema.org/Float", "characteristic_id": "char", "use_as_tag": true, "function_id": "fn", "aspect_ids": ["asp"], "unit": "x"}]}
		}`,
		"deprecated aspect_id instead of aspect_ids": `{
			"name": "a", "description": "d", "image": "img", "default_restart": true,
			"configs": [{"name": "c", "description": "cd", "type": "https://schema.org/Text", "default_value": {"b": 1, "a": [1, 2]}}],
			"output": {"name": "root", "type": "https://schema.org/StructuredValue", "characteristic_id": "",
				"sub_content_variables": [{"name": "t", "type": "https://schema.org/Float", "characteristic_id": "char", "use_as_tag": true, "function_id": "fn", "aspect_id": "asp"}]}
		}`,
	}
	for name, body := range same {
		if got := hash(body); got != expected {
			t.Errorf("%v: hash differs", name)
		}
	}

	different := map[string]string{
		"name":            `"name": "a"` + "|" + `"name": "b"`,
		"description":     `"description": "d"` + "|" + `"description": "e"`,
		"image":           `"image": "img"` + "|" + `"image": "img2"`,
		"default_restart": `"default_restart": true` + "|" + `"default_restart": false`,
		"config type":     `"https://schema.org/Text"` + "|" + `"https://schema.org/Integer"`,
		"config default":  `"b": 1` + "|" + `"b": 2`,
		"output type":     `"https://schema.org/StructuredValue"` + "|" + `"https://schema.org/Text"`,
		"sub variable":    `"characteristic_id": "char"` + "|" + `"characteristic_id": "char2"`,
		"use_as_tag":      `"use_as_tag": true` + "|" + `"use_as_tag": false`,
		"function":        `"function_id": "fn"` + "|" + `"function_id": "fn2"`,
		"aspect":          `"aspect_ids": ["asp"]` + "|" + `"aspect_ids": ["asp", "asp2"]`,
	}
	for name, replacement := range different {
		parts := strings.SplitN(replacement, "|", 2)
		if !strings.Contains(base, parts[0]) {
			t.Fatalf("%v: %v not in the test body", name, parts[0])
		}
		if got := hash(strings.Replace(base, parts[0], parts[1], 1)); got == expected {
			t.Errorf("%v: hash does not change", name)
		}
	}
}

// the token of the service is shared with the request paths; run with -race
func TestAdminAuthorizationIsGuarded(t *testing.T) {
	f := newResourceCheckFixture(t, model.ReleaseUsedResources{})
	wg := sync.WaitGroup{}
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			token, err := f.ctrl.adminAuthorization()
			if err != nil || token != fakeAdminToken {
				t.Errorf("%v, %v", token, err)
			}
		}()
	}
	wg.Wait()
}
