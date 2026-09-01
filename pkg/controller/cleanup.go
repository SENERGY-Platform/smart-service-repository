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
	"context"
	"net/http"

	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
)

func (this *Controller) Cleanup(ctx context.Context, ignoreModuleDeleteError bool) (result []error) {
	this.config.GetLogger().InfoContext(ctx, "start cleanup")
	this.cleanupMux.Lock()
	defer this.cleanupMux.Unlock()
	this.retryMarkedReleases(ctx)
	err := this.instanceCleanup(ctx)
	if err != nil {
		result = append(result, err...)
	}
	err = this.moduleCleanup(ctx, ignoreModuleDeleteError)
	if err != nil {
		result = append(result, err...)
	}
	err = this.variableCleanup(ctx)
	if err != nil {
		result = append(result, err...)
	}
	return result
}

func (this *Controller) instanceCleanup(ctx context.Context) (result []error) {
	instances, err := this.camunda.GetProcessInstanceList(ctx)
	if err != nil {
		return []error{err}
	}
	for _, instance := range instances {
		_, err, code := this.db.GetInstance(ctx, instance.BusinessKey, "")
		if err != nil && code == http.StatusNotFound {
			this.config.GetLogger().InfoContext(ctx, "found orphaned process-instance --> delete from camunda", "instanceId", instance.Id, "businessKey", instance.BusinessKey, "endTime", instance.EndTime, "error", err)
			err = this.camunda.DeleteInstance(ctx, instance)
		}
		if err != nil {
			result = append(result, err)
			this.config.GetLogger().ErrorContext(ctx, "unable to remove instance from camunda in cleanup", "instanceId", instance.Id, "businessKey", instance.BusinessKey, "error", err)
		}
	}
	return result
}

func (this *Controller) moduleCleanup(ctx context.Context, ignoreModuleDeleteError bool) (result []error) {
	offset := 0
	limit := 1000
	cache := map[string]bool{}
	for {
		modules, err, _ := this.db.ListAllModules(ctx, model.ModuleQueryOptions{
			Limit:  limit,
			Offset: offset,
			Sort:   "id.asc",
		})
		if err != nil {
			result = append(result, err)
			return result
		}
		for _, module := range modules {
			exists, checked := cache[module.InstanceId]
			if !checked {
				exists = true
				_, err, code := this.db.GetInstance(ctx, module.InstanceId, "")
				if err != nil && code == http.StatusNotFound {
					exists = false
					err = nil
				}
				if err != nil {
					result = append(result, err)
					this.config.GetLogger().ErrorContext(ctx, "unable to read instance for cleanup", "error", err)
				} else {
					cache[module.InstanceId] = exists
				}
			}
			if !exists {
				this.config.GetLogger().InfoContext(ctx, "found orphaned module --> remove", "moduleId", module.Id, "instanceId", module.InstanceId)
				err, _ = this.deleteModule(ctx, module, ignoreModuleDeleteError)
				if err != nil {
					result = append(result, err)
					this.config.GetLogger().ErrorContext(ctx, "unable to remove module", "error", err)
				}
			}
		}
		if len(modules) < limit {
			return result
		}
		offset = offset + limit
	}
}

func (this *Controller) variableCleanup(ctx context.Context) (result []error) {
	offset := 0
	limit := 1000
	cache := map[string]bool{}
	for {
		variables, err, _ := this.db.ListAllVariables(ctx, model.VariableQueryOptions{
			Limit:  limit,
			Offset: offset,
			Sort:   "name.asc",
		})
		if err != nil {
			result = append(result, err)
			return result
		}
		for _, variable := range variables {
			exists, checked := cache[variable.InstanceId]
			if !checked {
				exists = true
				_, err, code := this.db.GetInstance(ctx, variable.InstanceId, "")
				if err != nil && code == http.StatusNotFound {
					exists = false
					err = nil
				}
				if err != nil {
					result = append(result, err)
					this.config.GetLogger().ErrorContext(ctx, "unable to read instance for cleanup", "error", err)
				} else {
					cache[variable.InstanceId] = exists
				}
			}
			if !exists {
				this.config.GetLogger().InfoContext(ctx, "found orphaned variable --> remove", "instanceId", variable.InstanceId, "variableName", variable.Name)
				err, _ = this.db.DeleteVariable(ctx, variable.InstanceId, variable.UserId, variable.Name)
				if err != nil {
					result = append(result, err)
					this.config.GetLogger().ErrorContext(ctx, "unable to remove variable", "error", err)
				}
			}
		}
		if len(variables) < limit {
			return result
		}
		offset = offset + limit
	}
}
