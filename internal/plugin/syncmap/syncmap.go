/*
	Copyright © 2023, SAS Institute Inc., Cary, NC, USA.  All Rights Reserved.
	SPDX-License-Identifier: Apache-2.0
*/

package syncmap

import (
	"fmt"
	"sync"
)

type SyncMap[K comparable, V any] struct {
	syncMap map[K]*V
	lock    sync.Mutex
}

func New[K comparable, V any]() *SyncMap[K, V] {
	return &SyncMap[K, V]{
		syncMap: make(map[K]*V),
		lock:    sync.Mutex{},
	}
}

func (s *SyncMap[K, V]) Delete(key K) {
	s.lock.Lock()
	delete(s.syncMap, key)
	s.lock.Unlock()
}

func (s *SyncMap[K, V]) Set(key K, value *V) {
	if value == nil {
		s.Delete(key)
		return
	}

	s.lock.Lock()
	s.syncMap[key] = value
	s.lock.Unlock()
}

func (s *SyncMap[K, V]) Get(key K) (*V, error) {
	value, found := s.syncMap[key]
	if !found {
		return nil, fmt.Errorf("value not found for key: %v", key)
	}

	return value, nil
}
