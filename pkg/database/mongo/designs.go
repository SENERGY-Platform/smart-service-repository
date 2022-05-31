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
)

var DesignBson = getBsonFieldObject[model.SmartServiceDesign]()

func init() {
	CreateCollections = append(CreateCollections, func(db *Mongo) error {
		var err error
		collection := db.client.Database(db.config.MongoTable).Collection(db.config.MongoCollectionDesign)
		err = db.ensureCompoundIndex(collection, "design_id_name_index", true, true, DesignBson.Id, DesignBson.Name)
		if err != nil {
			return err
		}
		return nil
	})
}

func (this *Mongo) designCollection() *mongo.Collection {
	return this.client.Database(this.config.MongoTable).Collection(this.config.MongoCollectionDesign)
}

func (this *Mongo) GetDesign(id string, userId string) (result model.SmartServiceDesign, err error, code int) {
	ctx, _ := getTimeoutContext()
	temp := this.designCollection().FindOne(ctx, bson.M{DesignBson.Id: id, DesignBson.UserId: userId})
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

func (this *Mongo) SetDesign(element model.SmartServiceDesign) (error, int) {
	ctx, _ := getTimeoutContext()
	_, err := this.designCollection().ReplaceOne(
		ctx,
		bson.M{
			DesignBson.Id:     element.Id,
			DesignBson.UserId: element.UserId,
		},
		element,
		options.Replace().SetUpsert(true))
	if err != nil {
		return err, http.StatusInternalServerError
	}
	return nil, http.StatusOK
}

func (this *Mongo) DeleteDesign(id string, userId string) (error, int) {
	ctx, _ := getTimeoutContext()
	_, err := this.designCollection().DeleteOne(ctx, bson.M{
		DesignBson.Id:     id,
		DesignBson.UserId: userId,
	})
	if err != nil {
		return err, http.StatusInternalServerError
	}
	return nil, http.StatusOK
}

func (this *Mongo) ListDesigns(userId string, query model.DesignQueryOptions) (result []model.SmartServiceDesign, err error, code int) {
	opt := createFindOptions(query)
	ctx, _ := getTimeoutContext()
	cursor, err := this.designCollection().Find(ctx, bson.M{DesignBson.UserId: userId}, opt)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	return readCursorResult[model.SmartServiceDesign](ctx, cursor)
}
