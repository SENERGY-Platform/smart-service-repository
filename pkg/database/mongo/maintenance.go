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
	"go.mongodb.org/mongo-driver/bson"
)

func (this *Mongo) RemoveFromRunningMaintenanceIds(instanceId string, removeMaintenanceIds []string) error {
	ctx, _ := getTimeoutContext()
	_, err := this.instanceCollection().UpdateOne(ctx, bson.M{
		InstanceBson.Id: instanceId,
	}, bson.M{
		"$pull": bson.M{"running_maintenance_ids": bson.M{"$in": removeMaintenanceIds}},
	})
	return err
}

func (this *Mongo) AddToRunningMaintenanceIds(instanceId string, maintenanceId string) error {
	ctx, _ := getTimeoutContext()
	_, err := this.instanceCollection().UpdateOne(ctx, bson.M{
		InstanceBson.Id: instanceId,
	}, bson.M{
		"$push": bson.M{"running_maintenance_ids": maintenanceId},
	})
	return err
}
