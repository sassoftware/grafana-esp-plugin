/*
	Copyright © 2023, SAS Institute Inc., Cary, NC, USA.  All Rights Reserved.
	SPDX-License-Identifier: Apache-2.0
*/

package query

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"grafana-esp-plugin/internal/plugin/server"
	"net/url"
	"strconv"
	"strings"
)

type Query struct {
	ServerUrl           url.URL
	ProjectName         string
	CqName              string
	WindowName          string
	Fields              []string
	EventInterval       uint64
	MaxEvents           uint64
	AuthorizationHeader *string
}

func New(s server.Server, projectName string, cqName string, windowName string, interval uint64, maxEvents uint64, fields []string, authorizationHeader *string) *Query {
	return &Query{
		ServerUrl:           s.GetUrl(),
		ProjectName:         projectName,
		CqName:              cqName,
		WindowName:          windowName,
		EventInterval:       interval,
		MaxEvents:           maxEvents,
		Fields:              fields,
		AuthorizationHeader: authorizationHeader,
	}
}

func (q *Query) ToChannelPath() string {
	hashString := q.calcHashString()
	channelPath := fmt.Sprintf("stream/%s", hashString)
	return channelPath
}

func (q *Query) calcHashString() string {
	b := bytes.Join([][]byte{
		[]byte(q.ServerUrl.String()),
		[]byte(q.ProjectName),
		[]byte(q.CqName),
		[]byte(q.WindowName),
		[]byte(strconv.Itoa(int(q.EventInterval))),
		[]byte(strconv.Itoa(int(q.MaxEvents))),
		[]byte(strings.Join(q.Fields, "/")),
	}, []byte{0})
	hashSum := sha256.Sum256(b)

	return fmt.Sprintf("%x", hashSum)
}
