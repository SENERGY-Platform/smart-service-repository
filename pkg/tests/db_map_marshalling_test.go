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
	"context"
	"encoding/json"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/database/mongo"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/tests/mocks"
	"reflect"
	"sync"
	"testing"
)

func TestMapMarshalling(t *testing.T) {
	if CI {
		t.Skip("not in ci")
	}
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mongoUrl, err := mocks.Mongo(ctx, wg)
	if err != nil {
		t.Error(err)
		return
	}

	config, err := configuration.Load("../../config.json")
	if err != nil {
		t.Error(err)
		return
	}
	config.MongoUrl = mongoUrl

	m, err := mongo.New(config)
	if err != nil {
		t.Error(err)
		return
	}

	instance := model.SmartServiceInstance{
		SmartServiceInstanceInit: model.SmartServiceInstanceInit{
			SmartServiceInstanceInfo: model.SmartServiceInstanceInfo{
				Name:        "test",
				Description: "test",
			},
			Parameters: []model.SmartServiceParameter{
				{
					Id:         "foo",
					Value:      map[string]interface{}{"foo": "bar"},
					Label:      "foo",
					ValueLabel: "foo",
				},
			},
		},
		Id:        "test",
		UserId:    "test",
		DesignId:  "test",
		ReleaseId: "test",
	}

	err, _ = m.SetInstance(instance)
	if err != nil {
		t.Error(err)
		return
	}

	result, err, _ := m.GetInstance("test", "test")
	if err != nil {
		t.Error(err)
		return
	}

	expected := jsonNormalize(instance)
	actual := jsonNormalize(result)

	if !reflect.DeepEqual(expected, actual) {
		t.Error(expected, actual)
	}
}

func jsonNormalize(in interface{}) (out interface{}) {
	temp, _ := json.Marshal(in)
	json.Unmarshal(temp, &out)
	return
}
