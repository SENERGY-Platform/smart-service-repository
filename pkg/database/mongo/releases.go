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
	"net/http"
	"regexp"
	"runtime/debug"
	"strings"
	"time"

	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var ReleaseBson = getBsonFieldObject[model.SmartServiceReleaseExtended]()

const ReleaseBsonMarkedAsUnfinished = "marked_as_unfinished"
const ReleaseBsonMarkedAsDeleted = "marked_as_deleted"
const ReleaseBsonMarkedAtUnixTimestamp = "marked_at_unix_timestamp"

// paths into model.ReleaseUsedResources, which getBsonFieldObject does not reach because the field is a pointer
const ReleaseBsonUsedResources = "used_resources"
const ReleaseBsonUsedProcessModels = ReleaseBsonUsedResources + ".process_models"
const ReleaseBsonUsedFlows = ReleaseBsonUsedResources + ".flows"
const ReleaseBsonUsedImportTypes = ReleaseBsonUsedResources + ".import_types"

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
		collection := db.client.Database(db.config.MongoDatabase).Collection(db.config.MongoCollectionRelease)
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
		err = db.ensureIndex(collection, "release_used_process_models_index", ReleaseBsonUsedProcessModels, true, false)
		if err != nil {
			debug.PrintStack()
			return err
		}
		err = db.ensureIndex(collection, "release_used_flows_index", ReleaseBsonUsedFlows, true, false)
		if err != nil {
			debug.PrintStack()
			return err
		}
		err = db.ensureIndex(collection, "release_used_import_types_index", ReleaseBsonUsedImportTypes, true, false)
		if err != nil {
			debug.PrintStack()
			return err
		}
		return nil
	})
}

func (this *Mongo) releaseCollection() *mongo.Collection {
	return this.client.Database(this.config.MongoDatabase).Collection(this.config.MongoCollectionRelease)
}

func (this *Mongo) MarkReleaseAsFinished(ctx context.Context, id string) (err error) {
	ctx, _ = getTimeoutContext(ctx)
	_, err = this.releaseCollection().UpdateOne(ctx, bson.M{
		ReleaseBson.Id: id,
	}, bson.M{
		"$set": bson.M{ReleaseBsonMarkedAsUnfinished: false},
	})
	return err
}

func (this *Mongo) GetMarkedReleases(ctx context.Context) (markedAsDeleted []model.SmartServiceReleaseExtended, markedAsUnfinished []model.SmartServiceReleaseExtended, err error) {
	filter := bson.M{
		ReleaseBsonMarkedAtUnixTimestamp: bson.M{"$lt": time.Now().Add(-1 * this.config.MarkAgeLimit.GetDuration()).UnixMilli()},
		"$or": []interface{}{
			bson.M{ReleaseBsonMarkedAsDeleted: true},
			bson.M{ReleaseBsonMarkedAsUnfinished: true},
		},
	}
	ctx, _ = context.WithTimeout(context.WithoutCancel(ctx), time.Minute)
	cursor, err := this.releaseCollection().Find(ctx, filter)
	if err != nil {
		return markedAsDeleted, markedAsUnfinished, err
	}
	defer cursor.Close(context.Background())
	fullList, err, _ := readCursorResult[SmartServiceReleaseExtendedWithSyncMarks](ctx, cursor)
	if err != nil {
		return markedAsDeleted, markedAsUnfinished, err
	}
	for _, element := range fullList {
		if element.MarkedAsDeleted {
			markedAsDeleted = append(markedAsDeleted, element.SmartServiceReleaseExtended)
		} else if element.MarkedAsUnfinished {
			markedAsUnfinished = append(markedAsUnfinished, element.SmartServiceReleaseExtended)
		}
	}
	return markedAsDeleted, markedAsUnfinished, err

}

func (this *Mongo) SetRelease(ctx context.Context, element model.SmartServiceReleaseExtended, markAsUnfinished bool) (error, int) {
	//store release
	ctx, _ = getTimeoutContext(ctx)
	_, err := this.releaseCollection().ReplaceOne(
		ctx,
		bson.M{
			ReleaseBson.Id: element.Id,
		},
		SmartServiceReleaseExtendedWithSyncMarks{
			SmartServiceReleaseExtended: element,
			SyncMarks: SyncMarks{
				MarkedAtUnixTimestamp: time.Now().UnixMilli(),
				MarkedAsUnfinished:    markAsUnfinished,
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

func (this *Mongo) GetRelease(ctx context.Context, id string, withMarked bool) (result model.SmartServiceReleaseExtended, err error, code int) {
	ctx, _ = getTimeoutContext(ctx)
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
	if errors.Is(err, mongo.ErrNoDocuments) {
		return result, ErrReleaseNotFound, http.StatusNotFound
	}
	if err != nil {
		return
	}
	err = temp.Decode(&result)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return result, ErrReleaseNotFound, http.StatusNotFound
	}
	return result, nil, http.StatusOK
}

func (this *Mongo) DeleteRelease(ctx context.Context, id string) (error, int) {
	ctx, _ = getTimeoutContext(ctx)
	_, err := this.releaseCollection().DeleteMany(ctx, bson.M{
		ReleaseBson.Id: id,
	})
	if err != nil {
		return err, http.StatusInternalServerError
	}
	return nil, http.StatusOK
}

func (this *Mongo) MarlReleaseAsDeleted(ctx context.Context, id string) (error, int) {
	ctx, _ = getTimeoutContext(ctx)
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

func (this *Mongo) ListReleases(ctx context.Context, options model.ListReleasesOptions) (result []model.SmartServiceReleaseExtended, total int64, err error) {
	ctx, _ = getTimeoutContext(ctx)
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
		escapedSearch := regexp.QuoteMeta(search)
		filter = addAndFilter(filter, bson.M{
			"$or": []interface{}{
				bson.M{ReleaseBson.Name: bson.M{"$regex": escapedSearch, "$options": "i"}},
				bson.M{ReleaseBson.Description: bson.M{"$regex": escapedSearch, "$options": "i"}},
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
		return result, total, err
	}
	defer cursor.Close(context.Background())
	result, err, _ = readCursorResult[model.SmartServiceReleaseExtended](ctx, cursor)
	if err != nil {
		return result, total, err
	}
	total, err = this.releaseCollection().CountDocuments(ctx, filter)
	if err != nil {
		return result, total, err
	}
	return result, total, err
}

func (this *Mongo) GetReleasesByDesignId(ctx context.Context, designId string) (result []model.SmartServiceReleaseExtended, err error) {
	ctx, _ = getTimeoutContext(ctx)
	cursor, err := this.releaseCollection().Find(ctx, bson.M{ReleaseBson.DesignId: designId, ReleaseBsonMarkedAsDeleted: bson.M{"$ne": true}})
	if err != nil {
		return result, err
	}
	defer cursor.Close(context.Background())
	result, err, _ = readCursorResult[model.SmartServiceReleaseExtended](ctx, cursor)
	return result, err
}

func (this *Mongo) GetPreviousReleases(ctx context.Context, releaseId string) (result []model.SmartServiceReleaseExtended, err error) {
	ctx, _ = getTimeoutContext(ctx)
	cursor, err := this.releaseCollection().Find(ctx, bson.M{ReleaseBson.NewReleaseId: releaseId, ReleaseBsonMarkedAsDeleted: bson.M{"$ne": true}})
	if err != nil {
		return result, err
	}
	defer cursor.Close(context.Background())
	result, err, _ = readCursorResult[model.SmartServiceReleaseExtended](ctx, cursor)
	return result, err
}

// withoutUsedResourcesFilter matches releases that lack used_resources; the null match on an indexed path also matches a missing field
func withoutUsedResourcesFilter() bson.M {
	return bson.M{ReleaseBsonUsedProcessModels: nil}
}

const usedResourcesBackfillBatchSize = 10

// ForEachReleaseWithoutUsedResources calls f with id and bpmn_xml of every release not marked as deleted that lacks used_resources;
// the cursor reads in small batches, so the bpmn of all releases is never held at once. An error of f stops the iteration.
func (this *Mongo) ForEachReleaseWithoutUsedResources(ctx context.Context, f func(release model.SmartServiceReleaseExtended) error) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	filter := withoutUsedResourcesFilter()
	filter[ReleaseBsonMarkedAsDeleted] = bson.M{"$ne": true}
	opts := options.Find().
		SetProjection(bson.M{ReleaseBson.Id: 1, ReleaseBson.BpmnXml: 1}).
		SetBatchSize(usedResourcesBackfillBatchSize)
	cursor, err := this.releaseCollection().Find(ctx, filter, opts)
	if err != nil {
		return err
	}
	defer cursor.Close(context.Background())
	for cursor.Next(ctx) {
		release := model.SmartServiceReleaseExtended{}
		err = cursor.Decode(&release)
		if err != nil {
			return err
		}
		err = f(release)
		if err != nil {
			return err
		}
	}
	return cursor.Err()
}

// SetReleaseUsedResources sets used_resources only where it is still missing, so it never replaces a value stored meanwhile;
// updated reports whether this call wrote it
func (this *Mongo) SetReleaseUsedResources(ctx context.Context, id string, used model.ReleaseUsedResources) (updated bool, err error) {
	ctx, cancel := getTimeoutContext(ctx)
	defer cancel()
	// a nil list would be stored as null and make the release look unindexed forever
	for _, list := range []*[]string{&used.ProcessModels, &used.Flows, &used.ImportTypes} {
		if *list == nil {
			*list = []string{}
		}
	}
	filter := withoutUsedResourcesFilter()
	filter[ReleaseBson.Id] = id
	result, err := this.releaseCollection().UpdateOne(ctx, filter, bson.M{"$set": bson.M{ReleaseBsonUsedResources: used}})
	if err != nil {
		return false, err
	}
	return result.MatchedCount > 0, nil
}

func usedResourcesPath(kind model.ResourceKind) (string, error) {
	switch kind {
	case model.ResourceKindProcessModels:
		return ReleaseBsonUsedProcessModels, nil
	case model.ResourceKindFlows:
		return ReleaseBsonUsedFlows, nil
	case model.ResourceKindImportTypes:
		return ReleaseBsonUsedImportTypes, nil
	default:
		return "", errors.New("unknown resource kind")
	}
}

// ListReleasesUsingResource returns id, design_id, name and new_release_id of the releases not marked as deleted that use the resource
func (this *Mongo) ListReleasesUsingResource(ctx context.Context, kind model.ResourceKind, resourceId string) (result []model.SmartServiceRelease, err error) {
	path, err := usedResourcesPath(kind)
	if err != nil {
		return result, err
	}
	ctx, cancel := getTimeoutContext(ctx)
	defer cancel()
	projection := bson.M{ReleaseBson.Id: 1, ReleaseBson.DesignId: 1, ReleaseBson.Name: 1, ReleaseBson.NewReleaseId: 1}
	cursor, err := this.releaseCollection().Find(ctx, bson.M{
		path:                       resourceId,
		ReleaseBsonMarkedAsDeleted: bson.M{"$ne": true},
	}, options.Find().SetProjection(projection))
	if err != nil {
		return result, err
	}
	defer cursor.Close(context.Background())
	result, err, _ = readCursorResult[model.SmartServiceRelease](ctx, cursor)
	return result, err
}

// ListFinishedReleaseIds returns those of the given ids that name a stored release marked neither as deleted nor as unfinished
func (this *Mongo) ListFinishedReleaseIds(ctx context.Context, ids []string) (result []string, err error) {
	result = []string{}
	if len(ids) == 0 {
		return result, nil
	}
	ctx, cancel := getTimeoutContext(ctx)
	defer cancel()
	cursor, err := this.releaseCollection().Find(ctx, bson.M{
		ReleaseBson.Id:                bson.M{"$in": ids},
		ReleaseBsonMarkedAsDeleted:    bson.M{"$ne": true},
		ReleaseBsonMarkedAsUnfinished: bson.M{"$ne": true},
	}, options.Find().SetProjection(bson.M{ReleaseBson.Id: 1}))
	if err != nil {
		return result, err
	}
	defer cursor.Close(context.Background())
	releases, err, _ := readCursorResult[model.SmartServiceRelease](ctx, cursor)
	if err != nil {
		return result, err
	}
	for _, release := range releases {
		result = append(result, release.Id)
	}
	return result, nil
}
