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

import "github.com/SENERGY-Platform/smart-service-repository/pkg/model"

func (this *Mongo) SetModule(element model.SmartServiceModule) (error, int) {
	//TODO implement me
	panic("implement me")
}

func (this *Mongo) GetModule(id string, userId string) (model.SmartServiceModule, error, int) {
	//TODO implement me
	panic("implement me")
}

func (this *Mongo) ListModules(id string, query model.ModuleQueryOptions) ([]model.SmartServiceModule, error, int) {
	//TODO implement me
	panic("implement me")
}

func (this *Mongo) RemoveModulesOfInstance(instanceId string, userId string) (error, int) {
	//TODO implement me
	panic("implement me")
}
