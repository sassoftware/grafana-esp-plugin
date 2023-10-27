/*
	Copyright © 2023, SAS Institute Inc., Cary, NC, USA.  All Rights Reserved.
	SPDX-License-Identifier: Apache-2.0
*/

package messagedto

type InfoMessageDTO struct {
	Type string `json:"type"`
	Data struct {
		Discarded uint64 `json:"discarded"`
		Total     uint64 `json:"total"`
	} `json:"data"`
}
