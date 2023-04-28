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

import (
	"github.com/SENERGY-Platform/permission-search/lib/client"
	"github.com/SENERGY-Platform/permission-search/lib/model"
)

type QueryMessage = client.QueryMessage

type QueryFind = client.QueryFind

type QueryListIds = client.QueryListIds

type QueryCheckIds = client.QueryCheckIds

type QueryListCommons = client.QueryListCommons

type ListAfter = client.ListAfter

type QueryOperationType = client.QueryOperationType

const (
	QueryEqualOperation             = client.QueryEqualOperation
	QueryUnequalOperation           = client.QueryUnequalOperation
	QueryAnyValueInFeatureOperation = client.QueryAnyValueInFeatureOperation
)

type ConditionConfig = client.ConditionConfig

type Selection = client.Selection

type ResourceRights = model.ResourceRights

type ResourceRightsBase = model.ResourceRightsBase

type Right = model.Right
