/*
	Copyright © 2023, SAS Institute Inc., Cary, NC, USA.  All Rights Reserved.
	SPDX-License-Identifier: Apache-2.0
*/

package server

import (
	"fmt"
	"net/url"
	"path"
	"strconv"
)

type Server struct {
	url url.URL
}

func New(isTls bool, host string, portPtr *uint16, serverPath string) (*Server, error) {
	url, err := generateServerUrl(isTls, host, portPtr, serverPath)
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
	var portPtr *uint16 = nil
	if len(portString) > 0 {
		port, err := strconv.ParseUint(portString, 10, 16)
		if err != nil {
			return nil, err
		}
		port16 := uint16(port)
		portPtr = &port16
	}

	return New(isTls, host, portPtr, url.Path)
}

func (s *Server) GetUrl() url.URL {
	return s.url
}

func generateServerUrl(isTls bool, host string, portPtr *uint16, serverPath string) (*url.URL, error) {
	var urlScheme string
	if isTls {
		urlScheme = "wss"
	} else {
		urlScheme = "ws"
	}

	if portPtr != nil {
		host = fmt.Sprintf("%s:%d", host, *portPtr)
	}

	if len(serverPath) > 0 && serverPath[:1] == "/" {
		serverPath = serverPath[1:]
	}

	p := path.Join(serverPath, "eventStreamProcessing/v2/connect")

	urlString := fmt.Sprintf("%s://%s/%s", urlScheme, host, p)
	wsConnectionUrl, err := url.Parse(urlString)
	if err != nil {
		return nil, err
	}

	return wsConnectionUrl, nil
}
