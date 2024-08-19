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
	"strings"
	"time"
)

var ReleaseBson = getBsonFieldObject[model.SmartServiceReleaseExtended]()

const ReleaseBsonMarkedAsUnfinished = "marked_as_unfinished"
const ReleaseBsonMarkedAsDeleted = "marked_as_deleted"
const ReleaseBsonMarkedAtUnixTimestamp = "marked_at_unix_timestamp"

var ErrReleaseNotFound = errors.New("release not found")

type SyncMarks struct {
	MarkedAtUnixTimestamp int64 `json:"marked_at_unix_timestamp" bson:"marked_at_unix_timestamp"`
	MarkedAsUnfinished    bool  `json:"marked_as_unfinished" bson:"marked_as_unfinished"`
	MarkedAsDeleted       bool  `json:"marked_as_deleted" bson:"marked_as_deleted"`
}

type SmartServiceReleaseExtendedWithSyncMarks struct {
	model.SmartServiceReleaseExtended `bson:",inline"`
	SyncMarks                         `bson:",inline"`
}

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
		err = migrateReleasePermissions(db.config, collection)
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

func (this *Mongo) MarkReleaseAsFinished(id string) (err error) {
	ctx, _ := getTimeoutContext()
	_, err = this.releaseCollection().UpdateOne(ctx, bson.M{
		ReleaseBson.Id: id,
	}, bson.M{
		"$set": bson.M{ReleaseBsonMarkedAsUnfinished: false},
	})
	return err
}

func (this *Mongo) GetMarkedReleases() (markedAsDeleted []model.SmartServiceReleaseExtended, markedAsUnfinished []model.SmartServiceReleaseExtended, err error) {
	filter := bson.M{
		ReleaseBsonMarkedAtUnixTimestamp: bson.M{"$lt": time.Now().Add(-1 * this.config.MarkAgeLimit.GetDuration()).UnixMilli()},
		"$or": []interface{}{
			bson.M{ReleaseBsonMarkedAsDeleted: true},
			bson.M{ReleaseBsonMarkedAsUnfinished: true},
		},
	}
	ctx, _ := context.WithTimeout(context.Background(), time.Minute)
	cursor, err := this.releaseCollection().Find(ctx, filter)
	if err != nil {
		return markedAsDeleted, markedAsUnfinished, err
	}
	fullList, err, _ := readCursorResult[SmartServiceReleaseExtendedWithSyncMarks](ctx, cursor)
	if err != nil {
		return markedAsDeleted, markedAsUnfinished, err
	}
	for _, element := range fullList {
		if element.MarkedAsDeleted {
			markedAsDeleted = append(markedAsDeleted, element.SmartServiceReleaseExtended)
		} else if !element.MarkedAsUnfinished {
			markedAsUnfinished = append(markedAsUnfinished, element.SmartServiceReleaseExtended)
		}
	}
	return markedAsDeleted, markedAsUnfinished, err

}

func (this *Mongo) SetRelease(element model.SmartServiceReleaseExtended, markAsDone bool) (error, int) {
	//store release
	ctx, _ := getTimeoutContext()
	_, err := this.releaseCollection().ReplaceOne(
		ctx,
		bson.M{
			ReleaseBson.Id: element.Id,
		},
		SmartServiceReleaseExtendedWithSyncMarks{
			SmartServiceReleaseExtended: element,
			SyncMarks: SyncMarks{
				MarkedAtUnixTimestamp: time.Now().UnixMilli(),
				MarkedAsUnfinished:    markAsDone,
				MarkedAsDeleted:       false,
			},
		},
		options.Replace().SetUpsert(true))
	if err != nil {
		return err, http.StatusInternalServerError
	}

	//set instance new_release_id
	_, err = this.instanceCollection().UpdateMany(ctx, bson.M{
		InstanceBson.ReleaseId: element.Id,
	}, bson.M{
		"$set": bson.M{InstanceBson.NewReleaseId: element.NewReleaseId},
	})
	if err != nil {
		return err, http.StatusInternalServerError
	}

	return nil, http.StatusOK
}

func (this *Mongo) GetRelease(id string, withMarked bool) (result model.SmartServiceReleaseExtended, err error, code int) {
	ctx, _ := getTimeoutContext()
	filter := bson.M{
		ReleaseBson.Id:                id,
		ReleaseBsonMarkedAsDeleted:    bson.M{"$ne": true},
		ReleaseBsonMarkedAsUnfinished: bson.M{"$ne": true},
	}
	if withMarked {
		filter = bson.M{
			ReleaseBson.Id: id,
		}
	}
	temp := this.releaseCollection().FindOne(ctx, filter)
	err = temp.Err()
	if err == mongo.ErrNoDocuments {
		return result, ErrReleaseNotFound, http.StatusNotFound
	}
	if err != nil {
		return
	}
	err = temp.Decode(&result)
	if err == mongo.ErrNoDocuments {
		return result, ErrReleaseNotFound, http.StatusNotFound
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

func (this *Mongo) MarlReleaseAsDeleted(id string) (error, int) {
	ctx, _ := getTimeoutContext()
	_, err := this.releaseCollection().UpdateOne(ctx, bson.M{
		ReleaseBson.Id: id,
	}, bson.M{
		"$set": bson.M{ReleaseBsonMarkedAsDeleted: true},
	})
	if err != nil {
		return err, http.StatusInternalServerError
	}
	return err, http.StatusOK
}

func addAndFilter(filter bson.M, add bson.M) bson.M {
	andInterface, ok := filter["$and"]
	and := []interface{}{}
	if ok {
		and = andInterface.([]interface{})
	}
	and = append(and, add)
	filter["$and"] = and
	return filter
}

func (this *Mongo) ListReleases(options model.ListReleasesOptions) (result []model.SmartServiceReleaseExtended, err error) {
	ctx, _ := getTimeoutContext()
	opt := createFindOptions(options)
	filter := bson.M{
		ReleaseBsonMarkedAsDeleted:    bson.M{"$ne": true},
		ReleaseBsonMarkedAsUnfinished: bson.M{"$ne": true},
	}
	if options.InIds != nil {
		filter[ReleaseBson.Id] = bson.M{"$in": options.InIds}
	}
	search := strings.TrimSpace(options.Search)
	if search != "" {
		filter = addAndFilter(filter, bson.M{
			"$or": []interface{}{
				bson.M{ReleaseBson.Name: bson.M{"$regex": search, "$options": "i"}},
				bson.M{ReleaseBson.Description: bson.M{"$regex": search, "$options": "i"}},
			},
		})
	}
	if options.Latest {
		filter = addAndFilter(filter, bson.M{
			"$or": []interface{}{
				bson.M{ReleaseBson.NewReleaseId: ""},
				bson.M{ReleaseBson.NewReleaseId: bson.M{"$exists": false}},
			},
		})
	}
	cursor, err := this.releaseCollection().Find(ctx, filter, opt)
	if err != nil {
		return result, err
	}
	result, err, _ = readCursorResult[model.SmartServiceReleaseExtended](ctx, cursor)
	return result, err
}

func (this *Mongo) GetReleasesByDesignId(designId string) (result []model.SmartServiceReleaseExtended, err error) {
	ctx, _ := getTimeoutContext()
	cursor, err := this.releaseCollection().Find(ctx, bson.M{ReleaseBson.DesignId: designId, ReleaseBsonMarkedAsDeleted: bson.M{"$ne": true}})
	if err != nil {
		return result, err
	}
	result, err, _ = readCursorResult[model.SmartServiceReleaseExtended](ctx, cursor)
	return result, err
}
