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
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
)

// backfillTestDb implements only the methods the backfill uses; every other Database method panics
type backfillTestDb struct {
	Database
	releases  []model.SmartServiceReleaseExtended
	setErr    error
	mux       sync.Mutex
	active    int
	maxActive int
	stored    map[string]model.ReleaseUsedResources
}

func (this *backfillTestDb) ForEachReleaseWithoutUsedResources(ctx context.Context, f func(release model.SmartServiceReleaseExtended) error) error {
	this.mux.Lock()
	this.active++
	this.maxActive = max(this.maxActive, this.active)
	pending := []model.SmartServiceReleaseExtended{}
	for _, release := range this.releases {
		if _, ok := this.stored[release.Id]; !ok {
			pending = append(pending, release)
		}
	}
	this.mux.Unlock()
	defer func() {
		this.mux.Lock()
		this.active--
		this.mux.Unlock()
	}()
	time.Sleep(20 * time.Millisecond) // keeps the iteration open long enough for concurrent callers to overlap
	for _, release := range pending {
		if err := f(release); err != nil {
			return err
		}
	}
	return nil
}

func (this *backfillTestDb) SetReleaseUsedResources(ctx context.Context, id string, used model.ReleaseUsedResources) (bool, error) {
	if this.setErr != nil {
		return false, this.setErr
	}
	this.mux.Lock()
	defer this.mux.Unlock()
	if _, ok := this.stored[id]; ok {
		return false, nil
	}
	this.stored[id] = used
	return true, nil
}

func TestBackfillReleaseUsedResources(t *testing.T) {
	newDb := func() *backfillTestDb {
		return &backfillTestDb{
			stored: map[string]model.ReleaseUsedResources{},
			releases: []model.SmartServiceReleaseExtended{
				{SmartServiceRelease: model.SmartServiceRelease{Id: "ok"}, BpmnXml: usedResourcesTestBpmn(serviceTask("t1", "analytics", `<camunda:inputParameter name="analytics.flow_id">flow-1</camunda:inputParameter>`))},
				{SmartServiceRelease: model.SmartServiceRelease{Id: "broken"}, BpmnXml: "<bpmn:definitions"},
			},
		}
	}

	t.Run("unparsable bpmn is stored as empty and marked, not left for a retry", func(t *testing.T) {
		db := newDb()
		ctrl := &Controller{db: db}
		filled, unparsable, err := ctrl.BackfillReleaseUsedResources(context.Background())
		if err != nil || filled != 2 || unparsable != 1 {
			t.Fatalf("filled=%v unparsable=%v err=%v", filled, unparsable, err)
		}
		expected := model.ReleaseUsedResources{ProcessModels: []string{}, Flows: []string{}, ImportTypes: []string{}, Unparsable: true}
		if !reflect.DeepEqual(db.stored["broken"], expected) {
			t.Errorf("\nexpected %#v\ngot      %#v", expected, db.stored["broken"])
		}
		expected = model.ReleaseUsedResources{ProcessModels: []string{}, Flows: []string{"flow-1"}, ImportTypes: []string{}}
		if !reflect.DeepEqual(db.stored["ok"], expected) {
			t.Errorf("\nexpected %#v\ngot      %#v", expected, db.stored["ok"])
		}
	})

	t.Run("database error is returned", func(t *testing.T) {
		db := newDb()
		db.setErr = errors.New("test error")
		ctrl := &Controller{db: db}
		_, _, err := ctrl.BackfillReleaseUsedResources(context.Background())
		if !errors.Is(err, db.setErr) {
			t.Errorf("expected test error, got %v", err)
		}
	})

	t.Run("concurrent callers do not run their own backfill in parallel", func(t *testing.T) {
		db := newDb()
		ctrl := &Controller{db: db}
		wg := sync.WaitGroup{}
		mux := sync.Mutex{}
		totalFilled := 0
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				filled, _, err := ctrl.BackfillReleaseUsedResources(context.Background())
				if err != nil {
					t.Error(err)
				}
				mux.Lock()
				totalFilled += filled
				mux.Unlock()
			}()
		}
		wg.Wait()
		if db.maxActive != 1 {
			t.Errorf("expected one backfill at a time, got %v in parallel", db.maxActive)
		}
		if totalFilled != 2 {
			t.Errorf("expected 2 releases filled over all callers, got %v", totalFilled)
		}
	})
}
