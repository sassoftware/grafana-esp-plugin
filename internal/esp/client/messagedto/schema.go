/*
	Copyright © 2023, SAS Institute Inc., Cary, NC, USA.  All Rights Reserved.
	SPDX-License-Identifier: Apache-2.0
*/

package messagedto

type SchemaMessageDTO struct {
	SubscriptionId string `json:"@id"`
	WindowPath     string `json:"@window"`
	Fields         []struct {
		Key  string `json:"@key"`
		Name string `json:"@name"`
		Type string `json:"@type"`
	} `json:"fields"`
	SchemaString string `json:"schema-string"`
}
