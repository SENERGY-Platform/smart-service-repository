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

package model

import (
	"github.com/SENERGY-Platform/smart-service-repository/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/database"
	"github.com/google/uuid"
)

type SmartServiceDesign struct {
	Id          string `json:"id" bson:"id"`
	UserId      string `json:"user_id" bson:"user_id"`
	Name        string `json:"name" bson:"name"`
	Description string `json:"description"`
	BpmnXml     string `json:"bpmn_xml" bson:"bpmn_xml"`
	SvgXml      string `json:"svg_xml" bson:"svg_xml"`
}

func (this SmartServiceDesign) GetIndexInfo() (result []database.IndexInfo) {
	result = append(result, IndexInfo{
		Name:       "design_id_index",
		FieldNames: []string{"id", "user_id"},
		IsUnique:   true,
	})
	result = append(result, IndexInfo{
		Name:       "design_user_index",
		FieldNames: []string{"user_id"},
		IsUnique:   false,
	})
	result = append(result, IndexInfo{
		Name:       "design_name_index",
		FieldNames: []string{"name"},
		IsUnique:   false,
	})
	return result
}

func (this SmartServiceDesign) GetResourceName(config configuration.Config) string {
	return config.PersistenceResourceDesign
}

func (this SmartServiceDesign) GetIdMapping() map[string]interface{} {
	return map[string]interface{}{"id": this.Id, "user_id": this.UserId}
}

func (this *SmartServiceDesign) SetId() {
	this.Id = uuid.NewString()
}
