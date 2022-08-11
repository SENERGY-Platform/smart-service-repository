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

var ReleaseBson = getBsonFieldObject[model.SmartServiceReleaseExtended]()

func init() {
	CreateCollections = append(CreateCollections, func(db *Mongo) error {
		var err error
		collection := db.client.Database(db.config.MongoTable).Collection(db.config.MongoCollectionRelease)
		err = db.ensureIndex(collection, "release_id_index", ReleaseBson.Id, true, true)
		if err != nil {
			debug.PrintStack()
			return err
		}
		err = db.ensureIndex(collection, "release_design_index", ReleaseBson.DesignId, true, false)
		if err != nil {
			debug.PrintStack()
			return err
		}
		err = db.ensureIndex(collection, "release_creation_index", "created_at", true, false)
		if err != nil {
			debug.PrintStack()
			return err
		}
		return nil
	})
}

func (this *Mongo) releaseCollection() *mongo.Collection {
	return this.client.Database(this.config.MongoTable).Collection(this.config.MongoCollectionRelease)
}

func (this *Mongo) SetRelease(element model.SmartServiceReleaseExtended) (error, int) {
	//store release
	ctx, _ := getTimeoutContext()
	_, err := this.releaseCollection().ReplaceOne(
		ctx,
		bson.M{
			ReleaseBson.Id: element.Id,
		},
		element,
		options.Replace().SetUpsert(true))
	if err != nil {
		return err, http.StatusInternalServerError
	}

	//set instance new_release_id
	_, err = this.instanceCollection().UpdateMany(ctx, bson.M{
		InstanceBson.ReleaseId: element.Id,
	}, bson.M{
		"$set": bson.M{InstanceBson.NewReleaseId: element.Id},
	})
	if err != nil {
		return err, http.StatusInternalServerError
	}

	return nil, http.StatusOK
}

func (this *Mongo) SetReleaseError(id string, errMsg string) error {
	ctx, _ := getTimeoutContext()
	_, err := this.releaseCollection().UpdateOne(ctx, bson.M{
		ReleaseBson.Id: id,
	}, bson.M{
		"$set": bson.M{ReleaseBson.Error: errMsg},
	})
	return err
}

func (this *Mongo) GetRelease(id string) (result model.SmartServiceReleaseExtended, err error, code int) {
	ctx, _ := getTimeoutContext()
	temp := this.releaseCollection().FindOne(ctx, bson.M{ReleaseBson.Id: id})
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

func (this *Mongo) DeleteRelease(id string) (error, int) {
	ctx, _ := getTimeoutContext()
	_, err := this.releaseCollection().DeleteMany(ctx, bson.M{
		ReleaseBson.Id: id,
	})
	if err != nil {
		return err, http.StatusInternalServerError
	}
	return nil, http.StatusOK
}

func (this *Mongo) GetReleaseListByIds(ids []string, sort string) (result []model.SmartServiceReleaseExtended, err error) {
	ctx, _ := getTimeoutContext()
	opt := createFindOptions(model.ReleaseQueryOptions{Sort: sort})
	cursor, err := this.releaseCollection().Find(ctx, bson.M{ReleaseBson.Id: bson.M{"$in": ids}}, opt)
	if err != nil {
		return result, err
	}
	result, err, _ = readCursorResult[model.SmartServiceReleaseExtended](ctx, cursor)
	return result, err
}
