/*
	Copyright © 2023, SAS Institute Inc., Cary, NC, USA.  All Rights Reserved.
	SPDX-License-Identifier: Apache-2.0
*/

import { DataQuery, DataSourceJsonData } from '@grafana/data';

export interface EspQuery extends DataQuery {
  serverUrl: string | null;
  projectName: string | null;
  cqName: string | null;
  windowName: string | null;
  fields: string[];
}

export interface  Field {
  name: string;
  type: string;
}

export interface Window {
  name: string;
  fields: Field[]
}

export interface ContinuousQuery {
  name: string;
  windows: Window[];
}

export interface Project {
  name: string;
  continuousQueries: ContinuousQuery[];
}

export interface Server {
  url: string;
  name: string;
  projects: Project[];
}

export type EspObject = Server | Project | ContinuousQuery | Window | Field;
export enum EspObjectType {
  'SERVER',
  'PROJECT',
  'CONTINUOUS_QUERY',
  'WINDOW',
  'FIELD'
}

export function getEspObjectType(espObject: EspObject): EspObjectType {
  if (isServer(espObject)) {
    return EspObjectType.SERVER;
  } else if (isProject(espObject)) {
    return EspObjectType.PROJECT;
  } else if (isContinuousQuery(espObject)) {
    return EspObjectType.CONTINUOUS_QUERY;
  } else if (isWindow(espObject)) {
    return EspObjectType.WINDOW;
  } else if (isField(espObject)) {
    return EspObjectType.FIELD;
  } else {
    throw new Error(`Unrecognised EspObject type for: ${espObject}.`);
  }
}

export function isServer(espObject: EspObject | undefined): espObject is Server {
  return (espObject as Server).projects != null;
}

export function isProject(espObject: EspObject | undefined): espObject is Project {
  return (espObject as Project).continuousQueries != null;
}

export function isContinuousQuery(espObject: EspObject | undefined): espObject is ContinuousQuery {
  return (espObject as ContinuousQuery).windows != null;
}

export function isWindow(espObject: EspObject | undefined): espObject is Window {
  return (espObject as Window).fields != null;
}

export function isField(espObject: EspObject | undefined): espObject is Field {
  if (espObject == null) {
    return false;
  }

  const isOtherEspObject: boolean = isContinuousQuery(espObject) || isProject(espObject) || isServer(espObject) || isWindow(espObject);
  return !isOtherEspObject;
}


/**
 * These are options configured for each DataSource instance.
 */
export interface EspDataSourceOptions extends DataSourceJsonData {
  oauthPassThru: boolean;
}
