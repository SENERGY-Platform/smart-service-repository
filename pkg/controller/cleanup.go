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
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"log"
	"net/http"
)

func (this *Controller) Cleanup(ignoreModuleDeleteError bool) (result []error) {
	log.Println("start cleanup")
	this.cleanupMux.Lock()
	defer this.cleanupMux.Unlock()
	this.retryMarkedReleases()
	err := this.instanceCleanup()
	if err != nil {
		result = append(result, err...)
	}
	err = this.moduleCleanup(ignoreModuleDeleteError)
	if err != nil {
		result = append(result, err...)
	}
	err = this.variableCleanup()
	if err != nil {
		result = append(result, err...)
	}
	return result
}

func (this *Controller) instanceCleanup() (result []error) {
	instances, err := this.camunda.GetProcessInstanceList()
	if err != nil {
		return []error{err}
	}
	for _, instance := range instances {
		_, err, code := this.db.GetInstance(instance.BusinessKey, "")
		if err != nil && code == http.StatusNotFound {
			log.Println("found orphaned process-instance --> delete from camunda", instance.Id, instance.BusinessKey, instance.EndTime)
			err = this.camunda.DeleteInstance(instance)
		}
		if err != nil {
			result = append(result, err)
			log.Println("ERROR: unable to remove instance from camunda", err)
		}
	}
	return result
}

func (this *Controller) moduleCleanup(ignoreModuleDeleteError bool) (result []error) {
	offset := 0
	limit := 1000
	cache := map[string]bool{}
	for {
		modules, err, _ := this.db.ListAllModules(model.ModuleQueryOptions{
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
				_, err, code := this.db.GetInstance(module.InstanceId, "")
				if err != nil && code == http.StatusNotFound {
					exists = false
					err = nil
				}
				if err != nil {
					result = append(result, err)
					log.Println("ERROR: unable to read instance for cleanup", err)
				} else {
					cache[module.InstanceId] = exists
				}
			}
			if !exists {
				log.Println("found orphaned module --> remove", module.Id, module.InstanceId)
				err, _ = this.deleteModule(module, ignoreModuleDeleteError)
				if err != nil {
					result = append(result, err)
					log.Println("ERROR: unable to remove module", err)
				}
			}
		}
		if len(modules) < limit {
			return result
		}
		offset = offset + limit
	}
}

func (this *Controller) variableCleanup() (result []error) {
	offset := 0
	limit := 1000
	cache := map[string]bool{}
	for {
		variables, err, _ := this.db.ListAllVariables(model.VariableQueryOptions{
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
				_, err, code := this.db.GetInstance(variable.InstanceId, "")
				if err != nil && code == http.StatusNotFound {
					exists = false
					err = nil
				}
				if err != nil {
					result = append(result, err)
					log.Println("ERROR: unable to read instance for cleanup", err)
				} else {
					cache[variable.InstanceId] = exists
				}
			}
			if !exists {
				log.Println("found orphaned variable --> remove", variable.InstanceId, variable.Name)
				err, _ = this.db.DeleteVariable(variable.InstanceId, variable.UserId, variable.Name)
				if err != nil {
					result = append(result, err)
					log.Println("ERROR: unable to remove variable", err)
				}
			}
		}
		if len(variables) < limit {
			return result
		}
		offset = offset + limit
	}
}
