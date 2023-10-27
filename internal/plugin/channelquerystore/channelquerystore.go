/*
	Copyright © 2023, SAS Institute Inc., Cary, NC, USA.  All Rights Reserved.
	SPDX-License-Identifier: Apache-2.0
*/

package channelquerystore

import (
	"fmt"
	"grafana-esp-plugin/internal/plugin/query"
	"sync"
)

type ChannelQueryStore struct {
	channelQueryMap     map[string]*query.Query
	channelQueryMapLock sync.Mutex
}

func New() *ChannelQueryStore {
	return &ChannelQueryStore{
		channelQueryMap:     make(map[string]*query.Query),
		channelQueryMapLock: sync.Mutex{},
	}
}

func (s *ChannelQueryStore) Delete(channelPath string) {
	s.channelQueryMapLock.Lock()
	delete(s.channelQueryMap, channelPath)
	s.channelQueryMapLock.Unlock()
}

func (s *ChannelQueryStore) Store(channelPath string, q *query.Query) {
	s.channelQueryMapLock.Lock()
	s.channelQueryMap[channelPath] = q
	s.channelQueryMapLock.Unlock()
}

func (s *ChannelQueryStore) Load(channelPath string) (*query.Query, error) {
	q, found := s.channelQueryMap[channelPath]
	if !found {
		return nil, fmt.Errorf("query not found for channel: %v", channelPath)
	}

	return q, nil
}
