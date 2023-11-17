/*
	Copyright © 2023, SAS Institute Inc., Cary, NC, USA.  All Rights Reserved.
	SPDX-License-Identifier: Apache-2.0
*/

package syncmap

import (
	"testing"
)

func TestSetAndGetString(t *testing.T) {
	m := New[string, string]()
	inputValue := "bar"

	m.Set("foo", &inputValue)
	outputValuePtr, err := m.Get("foo")
	outputValue := *outputValuePtr
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if outputValue != inputValue {
		t.Errorf("expected %v, got %v", inputValue, outputValue)
	}
}

func TestGetDeletedString(t *testing.T) {
	s := New[string, string]()
	inputValue := "bar"
	s.Set("foo", &inputValue)

	s.Delete("foo")

	outputValuePtr, err := s.Get("foo")
	if err == nil {
		t.Errorf("expected non-nil error")
	}

	if outputValuePtr != nil {
		t.Errorf("expected %v, got %v", nil, outputValuePtr)
	}
}

func TestSetNilString(t *testing.T) {
	s := New[string, string]()

	s.Set("foo", nil)

	outputValuePtr, err := s.Get("foo")
	if err == nil {
		t.Errorf("expected non-nil error")
	}

	if outputValuePtr != nil {
		t.Errorf("expected %v, got %v", nil, outputValuePtr)
	}
}
