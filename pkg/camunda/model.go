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

package camunda

type ProcessDefinition struct {
	Id                string `json:"id,omitempty"`
	Key               string `json:"key,omitempty"`
	Category          string `json:"category,omitempty"`
	Description       string `json:"description,omitempty"`
	Name              string `json:"name,omitempty"`
	Version           int    `json:"Version,omitempty"`
	Resource          string `json:"resource,omitempty"`
	DeploymentId      string `json:"deploymentId,omitempty"`
	Diagram           string `json:"diagram,omitempty"`
	Suspended         bool   `json:"suspended,omitempty"`
	TenantId          string `json:"tenantId,omitempty"`
	VersionTag        string `json:"versionTag,omitempty"`
	HistoryTimeToLive int    `json:"historyTimeToLive,omitempty"`
}

type HistoricProcessInstance struct {
	Id          string `json:"id"`
	EndTime     string `json:"endTime"`
	BusinessKey string `json:"businessKey"`
}
