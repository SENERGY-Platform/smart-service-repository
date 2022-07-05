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
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
	"runtime/debug"
)

var ModuleBson = getBsonFieldObject[model.SmartServiceModule]()

func init() {
	CreateCollections = append(CreateCollections, func(db *Mongo) error {
		var err error
		collection := db.client.Database(db.config.MongoTable).Collection(db.config.MongoCollectionModule)
		err = db.ensureCompoundIndex(collection, "module_id_user_index", true, true, ModuleBson.Id, ModuleBson.UserId)
		if err != nil {
			debug.PrintStack()
			return err
		}
		err = db.ensureCompoundIndex(collection, "module_instance_user_index", true, false, ModuleBson.InstanceId, ModuleBson.UserId)
		if err != nil {
			debug.PrintStack()
			return err
		}
		err = db.ensureCompoundIndex(collection, "module_type_user_index", true, false, ModuleBson.ModuleType, ModuleBson.UserId)
		if err != nil {
			debug.PrintStack()
			return err
		}
		err = db.ensureIndex(collection, "module_user_index", ModuleBson.UserId, true, false)
		if err != nil {
			debug.PrintStack()
			return err
		}
		return nil
	})
}

func (this *Mongo) moduleCollection() *mongo.Collection {
	return this.client.Database(this.config.MongoTable).Collection(this.config.MongoCollectionModule)
}

func (this *Mongo) GetModule(id string, userId string) (result model.SmartServiceModule, err error, code int) {
	ctx, _ := getTimeoutContext()
	temp := this.moduleCollection().FindOne(ctx, bson.M{ModuleBson.Id: id, ModuleBson.UserId: userId})
	err = temp.Err()
	if err == mongo.ErrNoDocuments {
		return result, err, http.StatusNotFound
	}
	if err != nil {
		return
	}
	err = temp.Decode(&result)
	if err == mongo.ErrNoDocuments {
		return result, err, http.StatusNotFound
	}
	return result, nil, http.StatusOK
}

func (this *Mongo) SetModule(element model.SmartServiceModule) (error, int) {
	ctx, _ := getTimeoutContext()
	_, err := this.moduleCollection().ReplaceOne(
		ctx,
		bson.M{
			ModuleBson.Id:     element.Id,
			ModuleBson.UserId: element.UserId,
		},
		element,
		options.Replace().SetUpsert(true))
	if err != nil {
		return err, http.StatusInternalServerError
	}
	return nil, http.StatusOK
}

func (this *Mongo) DeleteModule(id string, userId string) (error, int) {
	ctx, _ := getTimeoutContext()
	_, err := this.moduleCollection().DeleteOne(ctx, bson.M{
		ModuleBson.Id:     id,
		ModuleBson.UserId: userId,
	})
	if err != nil {
		return err, http.StatusInternalServerError
	}
	return nil, http.StatusOK
}

func (this *Mongo) ListModules(userId string, query model.ModuleQueryOptions) (result []model.SmartServiceModule, err error, code int) {
	opt := createFindOptions(query)
	ctx, _ := getTimeoutContext()
	filter := bson.M{ModuleBson.UserId: userId}
	if query.InstanceIdFilter != nil {
		filter[ModuleBson.InstanceId] = *query.InstanceIdFilter
	}
	if query.TypeFilter != nil {
		filter[ModuleBson.ModuleType] = *query.TypeFilter
	}
	cursor, err := this.moduleCollection().Find(ctx, filter, opt)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	return readCursorResult[model.SmartServiceModule](ctx, cursor)
}

func (this *Mongo) ListAllModules(query model.ModuleQueryOptions) (result []model.SmartServiceModule, err error, code int) {
	opt := createFindOptions(query)
	ctx, _ := getTimeoutContext()
	filter := bson.M{}
	cursor, err := this.moduleCollection().Find(ctx, filter, opt)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	return readCursorResult[model.SmartServiceModule](ctx, cursor)
}

func (this *Mongo) RemoveModulesOfInstance(instanceId string, userId string) (error, int) {
	ctx, _ := getTimeoutContext()
	_, err := this.moduleCollection().DeleteOne(ctx, bson.M{
		ModuleBson.InstanceId: instanceId,
		ModuleBson.UserId:     userId,
	})
	if err != nil {
		return err, http.StatusInternalServerError
	}
	return nil, http.StatusOK
}
