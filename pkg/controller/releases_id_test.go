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
	"net/http"
	"testing"

	"github.com/SENERGY-Platform/smart-service-repository/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
)

// releaseIdTestDb answers GetRelease and GetDesign; every other Database method panics, so a create that goes on to save fails the test
type releaseIdTestDb struct {
	Database
	releaseCode int
	designAsked bool
}

func (this *releaseIdTestDb) GetRelease(ctx context.Context, id string, withMarked bool) (model.SmartServiceReleaseExtended, error, int) {
	switch this.releaseCode {
	case http.StatusOK:
		return model.SmartServiceReleaseExtended{SmartServiceRelease: model.SmartServiceRelease{Id: id, Creator: "other-user"}}, nil, http.StatusOK
	case http.StatusNotFound:
		return model.SmartServiceReleaseExtended{}, errors.New("not found"), http.StatusNotFound
	default:
		return model.SmartServiceReleaseExtended{}, errors.New("db down"), this.releaseCode
	}
}

func (this *releaseIdTestDb) GetDesign(ctx context.Context, id string, userId string) (model.SmartServiceDesign, error, int) {
	this.designAsked = true
	return model.SmartServiceDesign{}, errors.New("not found"), http.StatusNotFound
}

func TestCreateReleaseWithAnIdInUse(t *testing.T) {
	create := func(t *testing.T, releaseCode int, id string) (*releaseIdTestDb, int) {
		db := &releaseIdTestDb{releaseCode: releaseCode}
		ctrl := &Controller{db: db}
		_, err, code := ctrl.CreateRelease(context.Background(), auth.Token{}, model.SmartServiceRelease{Id: id, DesignId: "design"})
		if err == nil {
			t.Fatal("expected an error")
		}
		return db, code
	}

	t.Run("an id in use is refused before anything else", func(t *testing.T) {
		db, code := create(t, http.StatusOK, "taken")
		if code != http.StatusConflict {
			t.Errorf("code %v, want %v", code, http.StatusConflict)
		}
		if db.designAsked {
			t.Error("the design was read although the id is taken")
		}
	})

	t.Run("a free id goes on to the design", func(t *testing.T) {
		db, code := create(t, http.StatusNotFound, "free")
		if !db.designAsked || code != http.StatusBadRequest {
			t.Errorf("design asked %v, code %v; want the design check and %v", db.designAsked, code, http.StatusBadRequest)
		}
	})

	t.Run("a failing lookup does not create", func(t *testing.T) {
		db, code := create(t, http.StatusInternalServerError, "unknown")
		if code != http.StatusInternalServerError || db.designAsked {
			t.Errorf("code %v, design asked %v", code, db.designAsked)
		}
	})

	t.Run("without an id the lookup is skipped", func(t *testing.T) {
		db, code := create(t, http.StatusOK, "")
		if !db.designAsked || code != http.StatusBadRequest {
			t.Errorf("design asked %v, code %v", db.designAsked, code)
		}
	})
}
