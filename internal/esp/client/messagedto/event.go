/*
	Copyright © 2023, SAS Institute Inc., Cary, NC, USA.  All Rights Reserved.
	SPDX-License-Identifier: Apache-2.0
*/

package messagedto

type EventEntryDTO = map[string]any

type EventMessageDTO struct {
	SubscriptionId string          `json:"@id"`
	Entries        []EventEntryDTO `json:"entries"`
}
