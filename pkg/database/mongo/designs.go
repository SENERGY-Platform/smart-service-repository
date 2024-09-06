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
	"context"
	"errors"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
	"runtime/debug"
)

var DesignBson = getBsonFieldObject[model.SmartServiceDesign]()

var ErrDesignNotFound = errors.New("design not found")

func init() {
	CreateCollections = append(CreateCollections, func(db *Mongo) error {
		var err error
		collection := db.client.Database(db.config.MongoTable).Collection(db.config.MongoCollectionDesign)
		err = db.ensureCompoundIndex(collection, "design_id_user_index", true, true, DesignBson.Id, DesignBson.UserId)
		if err != nil {
			debug.PrintStack()
			return err
		}

		ctx, _ := getTimeoutContext()
		_, err = collection.Indexes().CreateOne(ctx, mongo.IndexModel{
			Keys: bson.D([]bson.E{
				{Key: DesignBson.UserId, Value: 1},
				{Key: DesignBson.Name, Value: "text"},
				{Key: DesignBson.Description, Value: "text"},
			}),
			Options: options.Index().SetName("design_search_with_user_compound_index"),
		})
		return err
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
		return result, ErrDesignNotFound, http.StatusNotFound
	}
	if err != nil {
		return
	}
	err = temp.Decode(&result)
	if err == mongo.ErrNoDocuments {
		return result, ErrDesignNotFound, http.StatusNotFound
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
	_, err := this.designCollection().DeleteMany(ctx, bson.M{
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
	filter := bson.M{DesignBson.UserId: userId}
	if query.Search != "" {
		filter["$text"] = bson.M{"$search": query.Search}
	}
	cursor, err := this.designCollection().Find(ctx, filter, opt)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	defer cursor.Close(context.Background())
	return readCursorResult[model.SmartServiceDesign](ctx, cursor)
}
