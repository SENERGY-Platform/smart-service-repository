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

package database

import (
	"github.com/SENERGY-Platform/smart-service-repository/pkg/configuration"
)

type Database interface {
	InitCollections(...CollectionInitElement) error
	GetConfig() configuration.Config
	Set(resource string, where map[string]interface{}, value interface{}) error
	Get(resource string, where map[string]interface{}, result ResultSetter) error
	Delete(resource string, where map[string]interface{}) error
	List(resource string, where map[string]interface{}, limit int64, offset int64, sort string, addToResultList ResultSetter) error
}

type Decoder = func(interface{}) error
type ResultSetter = func(Decoder) error

type Resource interface {
	CollectionInitElement
	Crud
}

type CollectionInitElement interface {
	GetIndexInfo() []IndexInfo
}

type IndexInfo interface {
	GetIndexName() string
	GetFieldNames() []string
	Unique() bool
}

type Crud interface {
	GetResourceName(config configuration.Config) string
	GetIdMapping() map[string]interface{} //{idFieldName: id}
}

func Set[T Crud](db Database, value T) (err error) {
	return db.Set(value.GetResourceName(db.GetConfig()), value.GetIdMapping(), value)
}

func Get[T Crud](db Database, value T) (result T, err error) {
	err = db.Get(result.GetResourceName(db.GetConfig()), value.GetIdMapping(), func(decoder Decoder) error {
		return decoder(&result)
	})
	return result, err
}

func Delete[T Crud](db Database, value T) (err error) {
	return db.Delete(value.GetResourceName(db.GetConfig()), value.GetIdMapping())
}

func List[T Crud](db Database, where map[string]interface{}, limit int64, offset int64, sort string) (result []T, err error) {
	v := new(T)
	resource := (*v).GetResourceName(db.GetConfig())
	var adder ResultSetter = func(decoder Decoder) error {
		e := new(T)
		err = decoder(e)
		if err != nil {
			return err
		}
		result = append(result, *e)
		return nil
	}
	err = db.List(resource, where, limit, offset, sort, adder)
	return result, err
}
