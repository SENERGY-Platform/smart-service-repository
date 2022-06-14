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

func (this *Mongo) SetModules(elements []model.SmartServiceModule) (error, int) {
	if len(elements) == 0 {
		return nil, http.StatusOK //nothing to add -> ok
	}
	if len(elements) == 1 {
		return this.SetModule(elements[0]) //no need for transactions
	}
	ctx, _ := getTimeoutContext()
	session, err := this.client.StartSession()
	if err != nil {
		return err, http.StatusInternalServerError
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, func(sessionContext mongo.SessionContext) (transactionResult interface{}, err error) {
		for _, element := range elements {
			transactionResult, err = this.moduleCollection().ReplaceOne(
				sessionContext,
				bson.M{
					ModuleBson.Id:     element.Id,
					ModuleBson.UserId: element.UserId,
				},
				element,
				options.Replace().SetUpsert(true))
			if err != nil {
				return transactionResult, err
			}
		}
		return transactionResult, err
	})
	if err != nil {
		return err, http.StatusInternalServerError
	}
	return nil, http.StatusOK
}
