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

func (this *resourceCheckTestDb) ForEachReleaseWithUsedResources(ctx context.Context, f func(release model.SmartServiceReleaseExtended) error) error {
	this.mux.Lock()
	releases := slices.Clone(this.releases)
	this.mux.Unlock()
	for _, release := range releases {
		if err := f(release); err != nil {
			return err
		}
	}
	return nil
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

// fakeRepo answers GET {prefix}/{id} from a map; ids without an entry are 404
type fakeRepo struct {
	*httptest.Server
	mux       sync.Mutex
	responses map[string]fakeResponse
	requests  int
	auth      []string
}

type fakeResponse struct {
	status int
	body   string
}

func newFakeRepo(t *testing.T) *fakeRepo {
	repo := &fakeRepo{responses: map[string]fakeResponse{}}
	repo.Server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		repo.mux.Lock()
		defer repo.mux.Unlock()
		repo.requests++
		repo.auth = append(repo.auth, request.Header.Get("Authorization"))
		response, ok := repo.responses[request.URL.Path]
		if !ok {
			response = fakeResponse{status: http.StatusNotFound}
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

func (this *fakeRepo) requestCount() int {
	this.mux.Lock()
	defer this.mux.Unlock()
	return this.requests
}

type resourceCheckFixture struct {
	t    *testing.T
	db   *resourceCheckTestDb
	ctrl *Controller
	pm   *fakeRepo
	flow *fakeRepo
	imp  *fakeRepo
	now  int64
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
	f.ctrl = &Controller{
		config: configuration.Config{
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

func (this *overlapTestDb) ForEachReleaseWithUsedResources(ctx context.Context, f func(release model.SmartServiceReleaseExtended) error) error {
	this.mux.Lock()
	this.active++
	this.maxActive = max(this.maxActive, this.active)
	this.mux.Unlock()
	time.Sleep(20 * time.Millisecond)
	this.mux.Lock()
	this.active--
	this.mux.Unlock()
	return this.resourceCheckTestDb.ForEachReleaseWithUsedResources(ctx, f)
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
