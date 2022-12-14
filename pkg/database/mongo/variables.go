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

var VariableBson = getBsonFieldObject[model.SmartServiceInstanceVariable]()

var ErrVariableNotFound = errors.New("variable not found")

func init() {
	CreateCollections = append(CreateCollections, func(db *Mongo) error {
		var err error
		collection := db.client.Database(db.config.MongoTable).Collection(db.config.MongoCollectionVariables)
		err = db.ensureCompoundIndex(collection, "variables_instance_name_user_index", true, true, VariableBson.InstanceId, VariableBson.UserId, VariableBson.Name)
		if err != nil {
			debug.PrintStack()
			return err
		}
		err = db.ensureCompoundIndex(collection, "variables_instance_user_index", true, false, VariableBson.InstanceId, VariableBson.UserId)
		if err != nil {
			debug.PrintStack()
			return err
		}
		return nil
	})
}

func (this *Mongo) variableCollection() *mongo.Collection {
	return this.client.Database(this.config.MongoTable).Collection(this.config.MongoCollectionVariables)
}

func (this *Mongo) GetVariable(instanceId string, userId string, variableName string) (result model.SmartServiceInstanceVariable, err error, code int) {
	ctx, _ := getTimeoutContext()
	temp := this.variableCollection().FindOne(ctx, bson.M{VariableBson.InstanceId: instanceId, VariableBson.UserId: userId, VariableBson.Name: variableName})
	err = temp.Err()
	if err == mongo.ErrNoDocuments {
		return result, ErrVariableNotFound, http.StatusNotFound
	}
	if err != nil {
		return
	}
	err = temp.Decode(&result)
	if err == mongo.ErrNoDocuments {
		return result, ErrVariableNotFound, http.StatusNotFound
	}
	return result, nil, http.StatusOK
}

func (this *Mongo) SetVariable(element model.SmartServiceInstanceVariable) (model.SmartServiceInstanceVariable, error, int) {
	instance, err, code := this.GetInstance(element.InstanceId, element.UserId)
	if err != nil {
		return element, err, code
	}
	element.InstanceId = instance.Id //replace possible running_maintenance_ids with instance id
	ctx, _ := getTimeoutContext()
	_, err = this.variableCollection().ReplaceOne(
		ctx,
		bson.M{
			VariableBson.InstanceId: element.InstanceId,
			VariableBson.UserId:     element.UserId,
			VariableBson.Name:       element.Name,
		},
		element,
		options.Replace().SetUpsert(true))
	if err != nil {
		return element, err, http.StatusInternalServerError
	}
	return element, nil, http.StatusOK
}

func (this *Mongo) DeleteVariable(instanceId string, userId string, variableName string) (error, int) {
	ctx, _ := getTimeoutContext()
	_, err := this.variableCollection().DeleteMany(ctx, bson.M{
		VariableBson.InstanceId: instanceId,
		VariableBson.UserId:     userId,
		VariableBson.Name:       variableName,
	})
	if err != nil {
		return err, http.StatusInternalServerError
	}
	return nil, http.StatusOK
}

func (this *Mongo) ListVariables(instanceId string, userId string, query model.VariableQueryOptions) (result []model.SmartServiceInstanceVariable, err error, code int) {
	opt := createFindOptions(query)
	ctx, _ := getTimeoutContext()
	filter := bson.M{VariableBson.InstanceId: instanceId, VariableBson.UserId: userId}
	cursor, err := this.variableCollection().Find(ctx, filter, opt)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	return readCursorResult[model.SmartServiceInstanceVariable](ctx, cursor)
}

func (this *Mongo) ListAllVariables(query model.VariableQueryOptions) (result []model.SmartServiceInstanceVariable, err error, code int) {
	opt := createFindOptions(query)
	ctx, _ := getTimeoutContext()
	filter := bson.M{}
	cursor, err := this.variableCollection().Find(ctx, filter, opt)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	return readCursorResult[model.SmartServiceInstanceVariable](ctx, cursor)
}

func (this *Mongo) RemoveVariablesOfInstance(instanceId string, userId string) (error, int) {
	ctx, _ := getTimeoutContext()
	_, err := this.variableCollection().DeleteMany(ctx, bson.M{
		VariableBson.InstanceId: instanceId,
		VariableBson.UserId:     userId,
	})
	if err != nil {
		return err, http.StatusInternalServerError
	}
	return nil, http.StatusOK
}
