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

import "strings"

type ModuleQueryOptions struct {
	TypeFilter       *string
	InstanceIdFilter *string
	Limit            int
	Offset           int
	Sort             string
}

func (this ModuleQueryOptions) GetLimit() int64 {
	return int64(this.Limit)
}

func (this ModuleQueryOptions) GetOffset() int64 {
	return int64(this.Offset)
}

func (this ModuleQueryOptions) GetSort() string {
	if this.Sort == "" {
		return "name.asc"
	}
	return this.Sort
}

type DesignQueryOptions struct {
	Limit  int
	Offset int
	Sort   string
}

func (this DesignQueryOptions) GetLimit() int64 {
	return int64(this.Limit)
}

func (this DesignQueryOptions) GetOffset() int64 {
	return int64(this.Offset)
}

func (this DesignQueryOptions) GetSort() string {
	if this.Sort == "" {
		return "name.asc"
	}
	return this.Sort
}

type ReleaseQueryOptions struct {
	Limit  int
	Offset int
	Sort   string
}

func (this ReleaseQueryOptions) GetLimit() int64 {
	return int64(this.Limit)
}

func (this ReleaseQueryOptions) GetOffset() int64 {
	return int64(this.Offset)
}

func (this ReleaseQueryOptions) GetSort() string {
	if this.Sort == "" {
		return "name.asc"
	}
	return this.Sort
}

func (this ReleaseQueryOptions) GetSortField() string {
	field := this.GetSort()
	field = strings.TrimSuffix(field, ".asc")
	field = strings.TrimSuffix(field, ".desc")
	return field
}

func (this ReleaseQueryOptions) GetSortAsc() bool {
	return strings.HasSuffix(this.GetSort(), ".desc")
}
