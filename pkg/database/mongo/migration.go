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
	"time"
)

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
	for cursor.Next(context.Background()) {
		element := SmartServiceReleaseExtendedWithSyncMarks{}
		err = cursor.Decode(&element)
		if err != nil {
			return err
		}
		rights, err := permSearchClient.GetRights(permissionsv2.InternalAdminToken, config.SmartServiceReleasePermissionsTopic, element.Id)
		if err != nil {
			return err
		}
		element.Creator = rights.Creator
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
		for user, right := range rights.UserRights {
			permissions.UserPermissions[user] = permmodel.PermissionsMap{
				Read:         right.Read,
				Write:        right.Write,
				Execute:      right.Execute,
				Administrate: right.Administrate,
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
		_, err, _ = permV2Client.SetPermission(permissionsv2.InternalAdminToken, config.SmartServiceReleasePermissionsTopic, element.Id, permissions)
		if err != nil {
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
