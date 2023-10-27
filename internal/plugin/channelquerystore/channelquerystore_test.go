/*
	Copyright © 2023, SAS Institute Inc., Cary, NC, USA.  All Rights Reserved.
	SPDX-License-Identifier: Apache-2.0
*/

package channelquerystore

import (
	"grafana-esp-plugin/internal/plugin/query"
	"grafana-esp-plugin/internal/plugin/server"
	"testing"
)

func newFakeQuery(t *testing.T) *query.Query {
	qs, err := server.New(false, "foo", 1234)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	return query.New(*qs, "project", "cq", "window")
}

func TestStoreAndLoadQuery(t *testing.T) {
	s := New()
	q := newFakeQuery(t)

	s.Store("foo", q)
	loadedQuery, err := s.Load("foo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if loadedQuery != q {
		t.Errorf("expected %v, got %v", &q, loadedQuery)
	}
}

func TestLoadDeletedQuery(t *testing.T) {
	s := New()
	q := newFakeQuery(t)
	s.Store("foo", q)

	s.Delete("foo")

	loadedQuery, err := s.Load("foo")
	if err == nil {
		t.Errorf("expected non-nil error")
	}

	if loadedQuery != nil {
		t.Errorf("expected %v, got %v", nil, q)
	}
}
