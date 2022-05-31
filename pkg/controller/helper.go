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

func NewProducerFactory[T Producer](f GenericProducerFactory[T]) (result ProducerFactory) {
	return FMap3(f, func(element T) Producer { return element })
}

func FMap3[I1 any, I2 any, I3 any, ResultType any, NewResultType any](f func(in1 I1, in2 I2, in3 I3) (ResultType, error), c func(ResultType) NewResultType) func(in1 I1, in2 I2, in3 I3) (NewResultType, error) {
	return func(in1 I1, in2 I2, in3 I3) (result NewResultType, err error) {
		temp, err := f(in1, in2, in3)
		if err != nil {
			return result, err
		}
		result = c(temp)
		return result, err
	}
}
