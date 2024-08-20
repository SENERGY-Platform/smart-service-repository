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

package mongo

import (
	"context"
	permissionsearch "github.com/SENERGY-Platform/permission-search/lib/client"
	permissionsv2 "github.com/SENERGY-Platform/permissions-v2/pkg/client"
	permmodel "github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/configuration"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"net/http"
	"runtime/debug"
	"time"
)

// TODO: will be removed once permissions-search removes smart-service-releases
func migrateReleasePermissions(config configuration.Config, releases *mongo.Collection) error {
	if config.PermissionsUrl == "" || config.PermissionsUrl == "-" {
		log.Printf("skip migrateReleasePermissions because config.PermissionsUrl = '%v'\n", config.PermissionsUrl)
		return nil
	}
	if config.PermissionsV2Url == "" || config.PermissionsV2Url == "-" {
		log.Printf("skip migrateReleasePermissions because config.PermissionsV2Url = '%v'\n", config.PermissionsV2Url)
		return nil
	}

	permSearchClient := permissionsearch.NewClient(config.PermissionsUrl)
	permV2Client := permissionsv2.New(config.PermissionsV2Url)

	ctx, _ := context.WithTimeout(context.Background(), time.Minute)
	opt := options.Find()
	opt.SetSort(bson.D{{ReleaseBson.Id, 1}})
	cursor, err := releases.Find(ctx, bson.M{ReleaseBsonMarkedAtUnixTimestamp: bson.M{"$exists": false}}, opt)
	if err != nil {
		return err
	}
	init := false
	for cursor.Next(context.Background()) {
		if !init {
			init = true
			topic := configuration.GetTopicDesc(config)
			_, err, code := permV2Client.GetTopic(permissionsv2.InternalAdminToken, topic.Id)
			if err != nil && code != http.StatusNotFound {
				return err
			}
			if code == http.StatusNotFound {
				_, err, _ := permV2Client.SetTopic(permissionsv2.InternalAdminToken, topic)
				if err != nil {
					return err
				}
			}
		}
		element := SmartServiceReleaseExtendedWithSyncMarks{}
		err = cursor.Decode(&element)
		if err != nil {
			return err
		}
		rights, err := permSearchClient.GetRights(permissionsv2.InternalAdminToken, config.SmartServiceReleasePermissionsTopic, element.Id)
		if err != nil {
			log.Println("ERROR: unable to get permission-search rights", config.PermissionsUrl, config.SmartServiceReleasePermissionsTopic, element.Id, err)
			debug.PrintStack()
			return err
		}
		element.Creator = rights.Creator
		if element.Creator == "" {
			element.Creator = "placeholder-for-invalid-data"
		}
		element.SyncMarks = SyncMarks{
			MarkedAtUnixTimestamp: time.Now().UnixMilli(),
			MarkedAsUnfinished:    false,
			MarkedAsDeleted:       false,
		}
		permissions := permissionsv2.ResourcePermissions{
			UserPermissions:  map[string]permmodel.PermissionsMap{},
			GroupPermissions: map[string]permmodel.PermissionsMap{},
			RolePermissions:  map[string]permmodel.PermissionsMap{},
		}
		hasAdminUser := false
		for user, right := range rights.UserRights {
			permissions.UserPermissions[user] = permmodel.PermissionsMap{
				Read:         right.Read,
				Write:        right.Write,
				Execute:      right.Execute,
				Administrate: right.Administrate,
			}
			if right.Administrate {
				hasAdminUser = true
			}
		}
		for group, right := range rights.GroupRights {
			permissions.RolePermissions[group] = permmodel.PermissionsMap{
				Read:         right.Read,
				Write:        right.Write,
				Execute:      right.Execute,
				Administrate: right.Administrate,
			}
		}
		if !hasAdminUser {
			permissions.UserPermissions[element.Creator] = permmodel.PermissionsMap{
				Read:         true,
				Write:        true,
				Execute:      true,
				Administrate: true,
			}
		}
		_, err, _ = permV2Client.SetPermission(permissionsv2.InternalAdminToken, config.SmartServiceReleasePermissionsTopic, element.Id, permissions)
		if err != nil {
			log.Println("ERROR: unable to set permissions-v2 permissions", config.SmartServiceReleasePermissionsTopic, element.Id, err)
			debug.PrintStack()
			return err
		}
		_, err = releases.ReplaceOne(ctx, bson.M{ReleaseBson.Id: element.Id}, element, options.Replace().SetUpsert(true))
		if err != nil {
			return err
		}
	}
	err = cursor.Err()
	return err
}
