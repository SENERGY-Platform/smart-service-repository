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

package mongo

import (
	"errors"
	"strings"
	"testing"

	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

// the used_resources paths are constants, so they are checked here against the bson tags of the stored document
func TestReleaseBsonUsedResourcesPaths(t *testing.T) {
	release := SmartServiceReleaseExtendedWithSyncMarks{
		SmartServiceReleaseExtended: model.SmartServiceReleaseExtended{
			UsedResources: &model.ReleaseUsedResources{
				ProcessModels: []string{"pm"},
				Flows:         []string{"flow"},
				ImportTypes:   []string{"it"},
			},
		},
	}
	raw, err := bson.Marshal(release)
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]string{
		ReleaseBsonUsedProcessModels: "pm",
		ReleaseBsonUsedFlows:         "flow",
		ReleaseBsonUsedImportTypes:   "it",
	}
	for path, value := range expected {
		element, err := bson.Raw(raw).LookupErr(strings.Split(path, ".")...)
		if err != nil {
			t.Errorf("%v: %v", path, err)
			continue
		}
		list, ok := element.ArrayOK()
		if !ok {
			t.Errorf("%v is no array", path)
			continue
		}
		first, err := list.IndexErr(0)
		if err != nil || first.Value().StringValue() != value {
			t.Errorf("%v: expected [%v], got %v", path, value, list)
		}
	}
}

// a release without used resources must be stored without the field, since the backfill looks for its absence
func TestReleaseBsonWithoutUsedResources(t *testing.T) {
	raw, err := bson.Marshal(SmartServiceReleaseExtendedWithSyncMarks{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = bson.Raw(raw).LookupErr(ReleaseBsonUsedResources)
	if !errors.Is(err, bsoncore.ErrElementNotFound) {
		t.Errorf("expected no %v field, got %v", ReleaseBsonUsedResources, err)
	}
}
