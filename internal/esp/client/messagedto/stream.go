/*
	Copyright © 2023, SAS Institute Inc., Cary, NC, USA.  All Rights Reserved.
	SPDX-License-Identifier: Apache-2.0
*/

package messagedto

type StreamMessageDTO struct {
	Interval      uint64   `json:"interval,omitempty"`
	MaxEvents     uint64   `json:"maxevents,omitempty"`
	Pagesize      int      `json:"pagesize,omitempty"`
	Action        string   `json:"action"`
	Id            string   `json:"id"`
	Window        string   `json:"window"`
	Schema        bool     `json:"schema"`
	UpdateDeletes bool     `json:"update-deletes"`
	Format        string   `json:"format"`
	IncludeFields []string `json:"include"`
}
