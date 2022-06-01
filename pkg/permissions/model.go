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

package permissions

type QueryMessage struct {
	Resource           string         `json:"resource"`
	Find               *QueryFind     `json:"find"`
	ListIds            *QueryListIds  `json:"list_ids"`
	CheckIds           *QueryCheckIds `json:"check_ids"`
	TermAggregate      *string        `json:"term_aggregate"`
	TermAggregateLimit int            `json:"term_aggregate_limit"`
}

type QueryFind struct {
	QueryListCommons
	Search string     `json:"search"`
	Filter *Selection `json:"filter"`
}

type QueryListIds struct {
	QueryListCommons
	Ids []string `json:"ids"`
}

type QueryCheckIds struct {
	Ids    []string `json:"ids"`
	Rights string   `json:"rights"`
}

type QueryListCommons struct {
	Limit    int        `json:"limit"`
	Offset   int        `json:"offset"`
	After    *ListAfter `json:"after"`
	Rights   string     `json:"rights"`
	SortBy   string     `json:"sort_by"`
	SortDesc bool       `json:"sort_desc"`
}

type ListAfter struct {
	SortFieldValue interface{} `json:"sort_field_value"`
	Id             string      `json:"id"`
}

type QueryOperationType string

const (
	QueryEqualOperation             QueryOperationType = "=="
	QueryUnequalOperation           QueryOperationType = "!="
	QueryAnyValueInFeatureOperation QueryOperationType = "any_value_in_feature"
)

type ConditionConfig struct {
	Feature   string             `json:"feature"`
	Operation QueryOperationType `json:"operation"`
	Value     interface{}        `json:"value"`
	Ref       string             `json:"ref"`
}

type Selection struct {
	And       []Selection     `json:"and"`
	Or        []Selection     `json:"or"`
	Not       *Selection      `json:"not"`
	Condition ConditionConfig `json:"condition"`
}
