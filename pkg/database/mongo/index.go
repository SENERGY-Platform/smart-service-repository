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
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx"
	"reflect"
	"strings"
)

func getBsonFieldObject[T any]() T {
	v := new(T)
	err := fillObjectWithItsBsonFieldNames(v, nil)
	if err != nil {
		panic(err)
	}
	return *v
}

func fillObjectWithItsBsonFieldNames(ptr interface{}, prefix []string) error {
	ptrval := reflect.ValueOf(ptr)
	objval := reflect.Indirect(ptrval)
	objecttype := objval.Type()
	for i := 0; i < objecttype.NumField(); i++ {
		field := objecttype.Field(i)
		if field.Type.Kind() == reflect.String {
			tags, err := bsoncodec.DefaultStructTagParser.ParseStructTags(field)
			if err != nil {
				return err
			}
			objval.Field(i).SetString(strings.Join(append(prefix, tags.Name), "."))
		}
		if field.Type.Kind() == reflect.Struct {
			tags, err := bsoncodec.DefaultStructTagParser.ParseStructTags(field)
			if err != nil {
				return err
			}
			if tags.Inline {
				err = fillObjectWithItsBsonFieldNames(objval.Field(i).Addr().Interface(), prefix)
			} else {
				err = fillObjectWithItsBsonFieldNames(objval.Field(i).Addr().Interface(), append(prefix, tags.Name))
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func getBsonFieldPath(obj interface{}, path string) (bsonPath string, err error) {
	t := reflect.TypeOf(obj)
	pathParts := strings.Split(path, ".")
	bsonPathParts := []string{}
	for _, name := range pathParts {
		field, found := t.FieldByName(name)
		if !found {
			return "", errors.New("field path '" + path + "' not found at '" + name + "'")
		}
		tags, err := bsoncodec.DefaultStructTagParser.ParseStructTags(field)
		if err != nil {
			return bsonPath, err
		}
		bsonPathParts = append(bsonPathParts, tags.Name)
		t = field.Type
	}
	bsonPath = strings.Join(bsonPathParts, ".")
	return
}

func getBsonFieldName(obj interface{}, fieldName string) (bsonName string, err error) {
	field, found := reflect.TypeOf(obj).FieldByName(fieldName)
	if !found {
		return "", errors.New("field '" + fieldName + "' not found")
	}
	tags, err := bsoncodec.DefaultStructTagParser.ParseStructTags(field)
	return tags.Name, err
}

func (this *Mongo) ensureCompoundIndex(collection *mongo.Collection, indexname string, asc bool, unique bool, indexKeys ...string) error {
	ctx, _ := getTimeoutContext()
	var direction int32 = -1
	if asc {
		direction = 1
	}
	keys := []bsonx.Elem{}
	for _, key := range indexKeys {
		keys = append(keys, bsonx.Elem{Key: key, Value: bsonx.Int32(direction)})
	}
	_, err := collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bsonx.Doc(keys),
		Options: options.Index().SetName(indexname).SetUnique(unique),
	})
	return err
}

func (this *Mongo) ensureIndex(collection *mongo.Collection, indexname string, indexKey string, asc bool, unique bool) error {
	ctx, _ := getTimeoutContext()
	var direction int32 = -1
	if asc {
		direction = 1
	}
	_, err := collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bsonx.Doc{{indexKey, bsonx.Int32(direction)}},
		Options: options.Index().SetName(indexname).SetUnique(unique),
	})
	return err
}

func (this *Mongo) ensureTextIndex(collection *mongo.Collection, indexname string, indexKeys ...string) error {
	if len(indexKeys) == 0 {
		return errors.New("expect at least one key")
	}
	keys := bsonx.Doc{}
	for _, key := range indexKeys {
		keys = append(keys, bsonx.Elem{Key: key, Value: bsonx.String("text")})
	}
	ctx, _ := getTimeoutContext()
	_, err := collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    keys,
		Options: options.Index().SetName(indexname),
	})
	return err
}
