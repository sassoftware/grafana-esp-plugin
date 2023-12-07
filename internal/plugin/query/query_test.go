/*
	Copyright © 2023, SAS Institute Inc., Cary, NC, USA.  All Rights Reserved.
	SPDX-License-Identifier: Apache-2.0
*/

package query

import (
	"grafana-esp-plugin/internal/plugin/server"
	"testing"
)

type equalityAssertion struct {
	Expected any
	Actual   any
}

func createQuery(t *testing.T) Query {
	s, err := server.FromUrlString("wss://host:12345")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	q := New(s.GetUrl(), "project", "cq", "window", 1, 2, []string{}, nil)

	return *q
}

func TestQueryToChannelPath(t *testing.T) {
	q1 := createQuery(t)

	q2 := createQuery(t)
	authHeader := "Bearer foo"
	q2.AuthorizationHeader = &authHeader

	q3 := createQuery(t)
	q3.ProjectName = "foo"

	equalityAssertions := []equalityAssertion{
		{"stream/408e4ac72d899e96e5d777c2e9e74939a6257a15d2b365d55221763589b03c4e", q1.ToChannelPath()},
		{"stream/408e4ac72d899e96e5d777c2e9e74939a6257a15d2b365d55221763589b03c4e", q2.ToChannelPath()},
		{"stream/025b476e61af885aa2b86c50a37da4411d92ac1cc6a8956171145ccb9812aca8", q3.ToChannelPath()},
	}

	for _, equalityAssertion := range equalityAssertions {
		expected := equalityAssertion.Expected
		actual := equalityAssertion.Actual

		if expected != actual {
			t.Errorf("expected %v, got %v", expected, actual)
		}
	}
}
