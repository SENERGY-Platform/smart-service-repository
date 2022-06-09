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

package auth

import (
	"encoding/json"
	"errors"
	"github.com/coocood/freecache"
	"log"
)

type Cache struct {
	l1                         *freecache.Cache
	defaultExpirationInSeconds int
}

type Item struct {
	Key   string
	Value []byte
}

var ErrNotFound = errors.New("key not found in cache")

var MB = 1024 * 1024 //used like NewCache(20 * MB, 60) to get a 20 mb cache

func NewCache(size int, defaultExpirationInSeconds int) *Cache {
	return &Cache{l1: freecache.NewCache(size), defaultExpirationInSeconds: defaultExpirationInSeconds}
}

func (this *Cache) get(key string) (value []byte, err error) {
	value, err = this.l1.Get([]byte(key))
	if err == freecache.ErrNotFound {
		err = ErrNotFound
	}
	return
}

func (this *Cache) set(key string, value []byte) {
	err := this.l1.Set([]byte(key), value, this.defaultExpirationInSeconds)
	if err != nil {
		log.Println("WARNING: err in cache.Set()", err)
	}
	return
}

func (this *Cache) setWithExpiration(key string, value []byte, expirationInSec int) {
	if expirationInSec <= 0 {
		expirationInSec = this.defaultExpirationInSeconds
	}
	err := this.l1.Set([]byte(key), value, expirationInSec)
	if err != nil {
		log.Println("WARNING: err in cache.setWithExpiration()", err)
	}
	return
}

func (this *Cache) Use(key string, getter func() (interface{}, error), result interface{}) (err error) {
	value, err := this.get(key)
	if err == nil {
		err = json.Unmarshal(value, result)
		return
	} else if err != ErrNotFound {
		log.Println("WARNING: err in cache.Get()", err)
	}
	temp, err := getter()
	if err != nil {
		return err
	}
	value, err = json.Marshal(temp)
	if err != nil {
		return err
	}
	this.set(key, value)
	return json.Unmarshal(value, &result)
}

func (this *Cache) UseWithExpirationInResult(key string, getter func() (interface{}, int, error), result interface{}) (err error) {
	value, err := this.get(key)
	if err == nil {
		err = json.Unmarshal(value, result)
		return
	} else if err != ErrNotFound {
		log.Println("WARNING: err in cache.Get()", err)
	}
	temp, expiration, err := getter()
	if err != nil {
		return err
	}
	value, err = json.Marshal(temp)
	if err != nil {
		return err
	}
	this.setWithExpiration(key, value, expiration)
	return json.Unmarshal(value, &result)
}
