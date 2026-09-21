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

import "github.com/SENERGY-Platform/models/go/models"

// The shapes of the device-selection selectables answer are defined in the shared model, and
// they are aliased here instead of copied. A copy silently drops whatever the answer gained
// since it was written - encoding/json discards a field the target struct does not declare,
// without a compiler error - which is how the aspect node lists of a path option would be
// lost here.
type (
	Selectable            = models.Selectable
	DeviceWithDisplayName = models.DeviceWithDisplayName
	Service               = models.Service
	DeviceGroup           = models.DeviceGroup
	Import                = models.Import
	ImportType            = models.ImportType
	PathOption            = models.PathOption
	Interaction           = models.Interaction
)
