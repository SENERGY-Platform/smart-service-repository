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

type MaintenanceProcedure struct {
	BpmnId                string                 `json:"bpmn_id" bson:"bpmn_id"`
	MessageRef            string                 `json:"message_ref" bson:"message_ref"`
	PublicEventId         string                 `json:"public_event_id" bson:"public_event_id"`
	InternalEventId       string                 `json:"internal_event_id" bson:"internal_event_id"`
	ParameterDescriptions []ParameterDescription `json:"parameter_descriptions" bson:"parameter_descriptions"`
}
