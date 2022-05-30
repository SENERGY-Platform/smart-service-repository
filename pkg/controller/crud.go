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

package controller

import "github.com/SENERGY-Platform/smart-service-repository/pkg/database"

type Crud = database.Crud
type DbProvider interface {
	GetDatabase() database.Database
}

func Set[T Crud](ctrl DbProvider, value T) (err error) {
	return database.Set(ctrl.GetDatabase(), value)
}

func Get[T Crud](ctrl DbProvider, value T) (result T, err error) {
	return database.Get(ctrl.GetDatabase(), value)
}

func Delete[T Crud](ctrl DbProvider, value T) (err error) {
	return database.Delete(ctrl.GetDatabase(), value)
}

func List[T Crud](ctrl DbProvider, where map[string]interface{}, limit int64, offset int64, sort string) (result []T, err error) {
	return database.List[T](ctrl.GetDatabase(), where, limit, offset, sort)
}
