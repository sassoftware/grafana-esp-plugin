/*
	Copyright © 2023, SAS Institute Inc., Cary, NC, USA.  All Rights Reserved.
	SPDX-License-Identifier: Apache-2.0
*/

package query

import (
	"testing"
)

type equalityAssertion struct {
	Expected any
	Actual   any
}

func createQuery(t *testing.T) Query {
	channelPath := "stream/wss/host/12345/project/cq/window"

	q, err := FromChannelPath(channelPath)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	return *q
}

func TestQueryFromChannelPath(t *testing.T) {
	q := createQuery(t)

	equalityAssertions := []equalityAssertion{
		{"wss", q.ServerUrl.Scheme},
		{"host:12345", q.ServerUrl.Host},
		{"project", q.ProjectName},
		{"cq", q.CqName},
		{"window", q.WindowName},
	}

	for _, equalityAssertion := range equalityAssertions {
		expected := equalityAssertion.Expected
		actual := equalityAssertion.Actual

		if expected != actual {
			t.Errorf("expected %v, got %v", expected, actual)
		}
	}
}

func TestQueryFromChannelPathError(t *testing.T) {
	invalidChannelPaths := []string{
		"foo/wss/host/12345/project/cq/window",
		"stream/wss/host/12345/project/cq",
		"stream/wss/host/12345/project/cq/window/foo",
		"stream/wss/host/65536/project/cq/window",
		"stream/wss/%%%/12345/project/cq/window",
	}

	for _, channelPath := range invalidChannelPaths {
		q, err := FromChannelPath(channelPath)
		if err == nil {
			t.Errorf("expected error, got %v", err)
		}

		if q != nil {
			t.Errorf("expected nil, got %v", q)
		}
	}
}

func TestQueryToChannelPath(t *testing.T) {
	q := createQuery(t)

	cp, err := q.ToChannelPath()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	equalityAssertions := []equalityAssertion{
		{"stream/wss/host/12345/project/cq/window", *cp},
	}

	for _, equalityAssertion := range equalityAssertions {
		expected := equalityAssertion.Expected
		actual := equalityAssertion.Actual

		if expected != actual {
			t.Errorf("expected %v, got %v", expected, actual)
		}
	}
}

func TestQueryToChannelPathError(t *testing.T) {
	q := createQuery(t)
	q.ProjectName = "../project"

	cp, err := q.ToChannelPath()
	if err == nil {
		t.Errorf("expected error, got: %v", err)
	}

	if cp != nil {
		t.Errorf("expected nil, got %v", cp)
	}
}
