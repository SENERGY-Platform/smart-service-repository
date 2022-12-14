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

package mongo

import (
	"errors"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
	"runtime/debug"
)

var InstanceBson = getBsonFieldObject[model.SmartServiceInstance]()

var ErrInstanceNotFound = errors.New("instance not found")

func init() {
	CreateCollections = append(CreateCollections, func(db *Mongo) error {
		var err error
		collection := db.client.Database(db.config.MongoTable).Collection(db.config.MongoCollectionInstance)
		err = db.ensureCompoundIndex(collection, "instance_id_user_index", true, true, InstanceBson.Id, InstanceBson.UserId)
		if err != nil {
			debug.PrintStack()
			return err
		}
		err = db.ensureIndex(collection, "instance_id_index", InstanceBson.Id, true, true)
		if err != nil {
			debug.PrintStack()
			return err
		}
		err = db.ensureIndex(collection, "instance_maintenance_ids_index", "running_maintenance_ids", true, false)
		if err != nil {
			debug.PrintStack()
			return err
		}
		err = db.ensureIndex(collection, "instance_release_index", InstanceBson.ReleaseId, true, false)
		if err != nil {
			debug.PrintStack()
			return err
		}
		return nil
	})
}

func (this *Mongo) instanceCollection() *mongo.Collection {
	return this.client.Database(this.config.MongoTable).Collection(this.config.MongoCollectionInstance)
}

func (this *Mongo) GetInstance(id string, userId string) (result model.SmartServiceInstance, err error, code int) {
	ctx, _ := getTimeoutContext()
	filter := bson.M{"$or": []interface{}{
		bson.M{InstanceBson.Id: id},
		bson.M{"running_maintenance_ids": id},
	}}
	if userId != "" {
		filter[InstanceBson.UserId] = userId
	}
	temp := this.instanceCollection().FindOne(ctx, filter)
	err = temp.Err()
	if err == mongo.ErrNoDocuments {
		return result, ErrInstanceNotFound, http.StatusNotFound
	}
	if err != nil {
		return
	}
	err = temp.Decode(&result)
	if err == mongo.ErrNoDocuments {
		return result, ErrInstanceNotFound, http.StatusNotFound
	}
	return result, nil, http.StatusOK
}

func (this *Mongo) SetInstance(element model.SmartServiceInstance) (error, int) {
	ctx, _ := getTimeoutContext()
	_, err := this.instanceCollection().ReplaceOne(
		ctx,
		bson.M{
			InstanceBson.Id:     element.Id,
			InstanceBson.UserId: element.UserId,
		},
		element,
		options.Replace().SetUpsert(true))
	if err != nil {
		return err, http.StatusInternalServerError
	}
	return nil, http.StatusOK
}

func (this *Mongo) DeleteInstance(id string, userId string) (err error, code int) {
	err, code = this.RemoveModulesOfInstance(id, userId)
	if err != nil {
		return err, code
	}
	err, code = this.RemoveVariablesOfInstance(id, userId)
	if err != nil {
		return err, code
	}
	ctx, _ := getTimeoutContext()
	_, err = this.instanceCollection().DeleteMany(ctx, bson.M{
		InstanceBson.Id:     id,
		InstanceBson.UserId: userId,
	})
	if err != nil {
		return err, http.StatusInternalServerError
	}
	return nil, http.StatusOK
}

func (this *Mongo) ListInstances(userId string, query model.InstanceQueryOptions) (result []model.SmartServiceInstance, err error, code int) {
	opt := createFindOptions(query)
	ctx, _ := getTimeoutContext()
	cursor, err := this.instanceCollection().Find(ctx, bson.M{InstanceBson.UserId: userId}, opt)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	return readCursorResult[model.SmartServiceInstance](ctx, cursor)
}

func (this *Mongo) SetInstanceError(id string, userId string, errMsg string) error {
	ctx, _ := getTimeoutContext()
	_, err := this.instanceCollection().UpdateOne(ctx, bson.M{
		InstanceBson.Id:     id,
		InstanceBson.UserId: userId,
	}, bson.M{
		"$set": bson.M{InstanceBson.Error: errMsg},
	})
	return err
}
