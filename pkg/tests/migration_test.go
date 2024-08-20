/*
 * Copyright 2024 InfAI (CC SES)
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
	"github.com/SENERGY-Platform/permissions-v2/pkg/client"
	permmodel "github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"github.com/SENERGY-Platform/service-commons/pkg/kafka"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/configuration"
	database "github.com/SENERGY-Platform/smart-service-repository/pkg/database/mongo"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/tests/docker"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
	"reflect"
	"sync"
	"testing"
	"time"
)

// TODO: will be removed once permissions-search removes smart-service-releases
func TestReleasePermissionsMigration(t *testing.T) {
	t.Log("will be removed once permissions-search removes smart-service-releases")
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config, err := configuration.Load("../../config.json")
	if err != nil {
		t.Error(err)
		return
	}
	config.MarkAgeLimit.SetDuration(time.Second)

	_, zkIp, err := docker.Zookeeper(ctx, wg)
	if err != nil {
		t.Error(err)
		return
	}
	zkUrl := zkIp + ":2181"

	kafkaUrl, err := docker.Kafka(ctx, wg, zkUrl)
	if err != nil {
		t.Error(err)
		return
	}
	time.Sleep(5 * time.Second)
	_, camundaPgIp, _, err := docker.Postgres(ctx, wg, "camunda")
	if err != nil {
		t.Error(err)
		return
	}

	config.CamundaUrl, err = docker.Camunda(ctx, wg, camundaPgIp, "5432")
	if err != nil {
		t.Error(err)
		return
	}

	_, searchIp, err := docker.OpenSearch(ctx, wg)
	if err != nil {
		t.Error(err)
		return
	}

	time.Sleep(time.Second)

	_, permIp, err := docker.PermSearch(ctx, wg, false, kafkaUrl, searchIp)
	if err != nil {
		t.Error(err)
		return
	}
	config.PermissionsUrl = "http://" + permIp + ":8080"

	_, mongoIp, err := docker.MongoDB(ctx, wg)
	if err != nil {
		t.Error(err)
		return
	}
	config.MongoUrl = "mongodb://" + mongoIp + ":27017"

	_, permV2Ip, err := docker.PermissionsV2(ctx, wg, config.MongoUrl)
	if err != nil {
		t.Error(err)
		return
	}
	config.PermissionsV2Url = "http://" + permV2Ip + ":8080"

	time.Sleep(5 * time.Second)

	producer, err := kafka.NewProducer(ctx, kafka.Config{
		KafkaUrl: kafkaUrl,
		Wg:       wg,
		OnError: func(err error) {
			t.Error(err)
		},
	}, config.SmartServiceReleasePermissionsTopic)
	if err != nil {
		t.Error(err)
		return
	}

	reg := bson.NewRegistryBuilder().RegisterTypeMapEntry(bsontype.EmbeddedDocument, reflect.TypeOf(bson.M{})).Build() //ensure map marshalling to interface
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(config.MongoUrl), options.Client().SetRegistry(reg))
	if err != nil {
		t.Error(err)
		return
	}

	element := model.SmartServiceRelease{
		Id:           "release-id",
		DesignId:     "design-id",
		Name:         "foo",
		NewReleaseId: "",
	}

	msg, err := json.Marshal(map[string]interface{}{
		"command": "PUT",
		"id":      "release-id",
		"release": element,
		"owner":   "test-migration-owner",
	})
	if err != nil {
		t.Error(err)
		return
	}

	err = producer.Produce("foo", msg)
	if err != nil {
		t.Error(err)
		return
	}

	collection := mongoClient.Database(config.MongoTable).Collection(config.MongoCollectionRelease)
	_, err = collection.ReplaceOne(
		ctx,
		bson.M{
			"id": element.Id,
		},
		element,
		options.Replace().SetUpsert(true))
	if err != nil {
		t.Error(err)
		return
	}

	element2 := database.SmartServiceReleaseExtendedWithSyncMarks{
		SmartServiceReleaseExtended: model.SmartServiceReleaseExtended{
			SmartServiceRelease: model.SmartServiceRelease{
				Id:           "release-id-2",
				DesignId:     "design-id-2",
				Name:         "foo2",
				NewReleaseId: "",
				Creator:      "test-migration-owner-2",
			},
		},
		SyncMarks: database.SyncMarks{
			MarkedAtUnixTimestamp: time.Now().UnixMilli(),
			MarkedAsUnfinished:    false,
			MarkedAsDeleted:       false,
		},
	}

	msg2, err := json.Marshal(map[string]interface{}{
		"command": "PUT",
		"id":      "release-id-2",
		"release": element2,
		"owner":   "test-migration-owner", //mismatch to element2.creator to test danger of overwrite
	})
	if err != nil {
		t.Error(err)
		return
	}

	err = producer.Produce("foo2", msg2)
	if err != nil {
		t.Error(err)
		return
	}

	_, err = collection.ReplaceOne(
		ctx,
		bson.M{
			"id": element2.Id,
		},
		element2,
		options.Replace().SetUpsert(true))
	if err != nil {
		t.Error(err)
		return
	}

	time.Sleep(5 * time.Second)

	db, err := database.New(config)
	if err != nil {
		t.Error(err)
		return
	}

	release, err, _ := db.GetRelease(element.Id, false)
	if err != nil {
		t.Error(err)
		return
	}
	if release.Id != element.Id {
		t.Error(release)
		return
	}
	if release.Name != element.Name {
		t.Error(release)
		return
	}
	if release.Creator != "test-migration-owner" {
		t.Error(release)
		return
	}

	release2, err, _ := db.GetRelease(element2.Id, false)
	if err != nil {
		t.Error(err)
		return
	}
	if release2.Id != element2.Id {
		t.Error(release)
		return
	}
	if release2.Name != element2.Name {
		t.Error(release)
		return
	}
	if release2.Creator != "test-migration-owner-2" {
		t.Error(release)
		return
	}

	deleted, unfinished, err := db.GetMarkedReleases()
	if err != nil {
		t.Error(err)
		return
	}
	if len(deleted) != 0 || len(unfinished) != 0 {
		t.Error(deleted, unfinished)
		return
	}

	permV2Client := client.New(config.PermissionsV2Url)
	permResource, err, _ := permV2Client.GetResource(client.InternalAdminToken, config.SmartServiceReleasePermissionsTopic, element.Id)
	if err != nil {
		t.Error(err)
		return
	}
	if !reflect.DeepEqual(permResource, client.Resource{
		Id:      element.Id,
		TopicId: config.SmartServiceReleasePermissionsTopic,
		ResourcePermissions: permmodel.ResourcePermissions{
			UserPermissions: map[string]permmodel.PermissionsMap{
				release.Creator: {
					Read:         true,
					Write:        true,
					Execute:      true,
					Administrate: true,
				},
			},
			GroupPermissions: map[string]permmodel.PermissionsMap{},
			RolePermissions: map[string]permmodel.PermissionsMap{
				"admin": {
					Read:         true,
					Write:        true,
					Execute:      true,
					Administrate: true,
				},
			},
		},
	}) {
		t.Error(permResource)
		return
	}

	_, _, code := permV2Client.GetResource(client.InternalAdminToken, config.SmartServiceReleasePermissionsTopic, element2.Id)
	if code != http.StatusNotFound {
		t.Error("element2 should not have been found for migration -> no migration -> no entry in permV2", code)
		return
	}

	_, err, _ = permV2Client.SetPermission(client.InternalAdminToken, config.SmartServiceReleasePermissionsTopic, element2.Id, client.ResourcePermissions{
		UserPermissions: map[string]permmodel.PermissionsMap{
			release2.Creator: {
				Read:         true,
				Write:        true,
				Execute:      true,
				Administrate: true,
			},
		},
	})
	if err != nil {
		t.Error(err)
		return
	}
}
