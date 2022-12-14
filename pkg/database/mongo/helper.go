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
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx"
	"net/http"
	"strings"
)

type QueryOptions interface {
	GetLimit() int64
	GetOffset() int64
	GetSort() string
}

func createFindOptions(query QueryOptions) *options.FindOptions {
	opt := options.Find()
	if query.GetLimit() > 0 {
		opt.SetLimit(query.GetLimit())
	}
	opt.SetSkip(query.GetOffset())

	sortby := query.GetSort()
	sortby = strings.TrimSuffix(sortby, ".asc")
	sortby = strings.TrimSuffix(sortby, ".desc")

	direction := int32(1)
	if strings.HasSuffix(query.GetSort(), ".desc") {
		direction = int32(-1)
	}
	opt.SetSort(bsonx.Doc{{sortby, bsonx.Int32(direction)}})
	return opt
}

func readCursorResult[T any](ctx context.Context, cursor *mongo.Cursor) (result []T, err error, code int) {
	result = []T{}
	for cursor.Next(ctx) {
		element := new(T)
		err = cursor.Decode(element)
		if err != nil {
			return result, err, http.StatusInternalServerError
		}
		result = append(result, *element)
	}
	err = cursor.Err()
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	return result, nil, http.StatusOK
}
