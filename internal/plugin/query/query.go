/*
	Copyright © 2023, SAS Institute Inc., Cary, NC, USA.  All Rights Reserved.
	SPDX-License-Identifier: Apache-2.0
*/

package query

import (
	"errors"
	"fmt"
	"grafana-esp-plugin/internal/plugin/server"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

const CHANNEL_PATH_REGEX_PATTERN string = `^[A-z0-9_\-/=.]*$`

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

func FromChannelPath(channelPath string) (*Query, error) {
	splitChannelPath := strings.Split(channelPath, "/")
	if len(splitChannelPath) < 7 {
		return nil, errors.New("invalid stream channel path length received")
	}
	channelType := splitChannelPath[0]
	if channelType != "stream" {
		return nil, errors.New("invalid stream channel path type received")
	}
	scheme := splitChannelPath[1]
	host := splitChannelPath[2]
	portString := splitChannelPath[3]
	projectName := splitChannelPath[4]
	cqName := splitChannelPath[5]
	windowName := splitChannelPath[6]
	intervalString := splitChannelPath[7]
	maxEventsString := splitChannelPath[8]
	fields := splitChannelPath[9:]

	port, err := strconv.ParseUint(portString, 10, 16)
	if err != nil {
		return nil, err
	}
	isTls := scheme == "wss"
	server, err := server.New(isTls, host, uint16(port))
	if err != nil {
		return nil, err
	}
	serverUrl := server.GetUrl()
	interval, err := strconv.ParseUint(intervalString, 10, 64)
	if err != nil {
		return nil, err
	}
	maxEvents, err := strconv.ParseUint(maxEventsString, 10, 64)
	if err != nil {
		return nil, err
	}

	return &Query{
		ServerUrl:     serverUrl,
		ProjectName:   projectName,
		CqName:        cqName,
		WindowName:    windowName,
		EventInterval: interval,
		MaxEvents:     maxEvents,
		Fields:        fields,
	}, nil
}

func (q Query) ToChannelPath() (*string, error) {
	channelPath := fmt.Sprintf("stream/%s/%s/%s/%s/%s/%s/%d/%d/%s",
		url.PathEscape(q.ServerUrl.Scheme),
		url.PathEscape(q.ServerUrl.Hostname()),
		q.ServerUrl.Port(),
		url.PathEscape(q.ProjectName),
		url.PathEscape(q.CqName),
		url.PathEscape(q.WindowName),
		q.EventInterval,
		q.MaxEvents,
		strings.Join(q.Fields, "/"),
	)

	// The Channel class depends on its path matching this arbitrary regex, so validate it here and prevent silent failures.
	if !regexp.MustCompile(CHANNEL_PATH_REGEX_PATTERN).MatchString(channelPath) {
		return nil, fmt.Errorf(`channel path "%s" must match %s pattern`, channelPath, CHANNEL_PATH_REGEX_PATTERN)
	}

	return &channelPath, nil
}
