/*
	Copyright © 2023, SAS Institute Inc., Cary, NC, USA.  All Rights Reserved.
	SPDX-License-Identifier: Apache-2.0
*/

package server

import (
	"fmt"
	"net/url"
	"strconv"
)

type Server struct {
	url url.URL
}

func New(isTls bool, host string, port uint16) (*Server, error) {
	url, err := generateServerUrl(isTls, host, port)
	if err != nil {
		return nil, err
	}

	return &Server{*url}, nil
}

func FromUrlString(urlString string) (*Server, error) {
	url, err := url.Parse(urlString)
	if err != nil {
		return nil, err
	}

	isTls := url.Scheme == "wss"
	host := url.Hostname()
	portString := url.Port()
	port, err := strconv.ParseUint(portString, 10, 16)
	if err != nil {
		return nil, err
	}

	return New(isTls, host, uint16(port))
}

func (s *Server) GetUrl() url.URL {
	return s.url
}

func generateServerUrl(isTls bool, host string, port uint16) (*url.URL, error) {
	var urlScheme string
	if isTls {
		urlScheme = "wss"
	} else {
		urlScheme = "ws"
	}

	var urlString string = fmt.Sprintf("%s://%s:%d/eventStreamProcessing/v2/connect", urlScheme, host, port)
	wsConnectionUrl, err := url.Parse(urlString)
	if err != nil {
		return nil, err
	}

	return wsConnectionUrl, nil
}
