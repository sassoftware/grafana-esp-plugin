/*
	Copyright © 2023, SAS Institute Inc., Cary, NC, USA.  All Rights Reserved.
	SPDX-License-Identifier: Apache-2.0
*/

package messagedto

type MessageDTO struct {
	Bulk           *[]string                 `json:"bulk"`
	Error          *ErrorMessageDTO          `json:"error"`
	Events         *EventMessageDTO          `json:"events"`
	ProjectLoaded  *ProjectLoadedMessageDTO  `json:"project-loaded"`
	ProjectRemoved *ProjectRemovedMessageDTO `json:"project-removed"`
	Schema         *SchemaMessageDTO         `json:"schema"`
	Info           *InfoMessageDTO           `json:"info"`
}
