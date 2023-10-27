/*
	Copyright © 2023, SAS Institute Inc., Cary, NC, USA.  All Rights Reserved.
	SPDX-License-Identifier: Apache-2.0
*/

package querydto

type QueryDTO struct {
	ServerUrl     string   `json:"serverUrl"`
	ProjectName   string   `json:"projectName"`
	CqName        string   `json:"cqName"`
	WindowName    string   `json:"windowName"`
	Fields        []string `json:"fields,omitempty"`
	Interval      uint64   `json:"intervalMs,omitempty"`
	MaxDataPoints uint64   `json:"maxDataPoints,omitempty"`
}
