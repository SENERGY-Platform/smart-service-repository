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

import (
	"fmt"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/permissions"
)

type PermSearchCharacteristicsWrapper struct {
	Raw model.Characteristic `json:"raw"`
}

func (this *Controller) GetCharacteristic(token string, id string) (characteristic *model.Characteristic, err error) {
	characteristicList := []PermSearchCharacteristicsWrapper{}
	err, code := this.permissions.Query(token, permissions.QueryMessage{
		Resource: this.config.KafkaCharacteristicsTopic,
		ListIds: &permissions.QueryListIds{
			QueryListCommons: permissions.QueryListCommons{
				Limit:  1,
				Offset: 0,
				Rights: "r",
				SortBy: "name",
			},
			Ids: []string{id},
		},
	}, &characteristicList)
	if err != nil {
		return nil, fmt.Errorf("unexpected permissions search query response for %v %v (%v: %w)", this.config.KafkaCharacteristicsTopic, id, code, err)
	}
	if len(characteristicList) != 1 {
		return nil, fmt.Errorf("unexpected permissions search query response for %v %v (expect 1 element, got %v)", this.config.KafkaCharacteristicsTopic, id, len(characteristicList))
	}
	return &characteristicList[0].Raw, nil
}
