/*
	Copyright © 2023, SAS Institute Inc., Cary, NC, USA.  All Rights Reserved.
	SPDX-License-Identifier: Apache-2.0
*/

import {DataQueryError, DataQueryRequest, DataQueryResponse, DataSourceInstanceSettings} from '@grafana/data';
import { DataSourceWithBackend } from '@grafana/runtime';
import { EspDataSourceOptions, EspQuery } from './types';
import { Observable } from 'rxjs';
import { map } from 'rxjs/operators';

export class DataSource extends DataSourceWithBackend<EspQuery, EspDataSourceOptions> {
  constructor(instanceSettings: DataSourceInstanceSettings<EspDataSourceOptions>) {
    super(instanceSettings);
  }

  query(options: DataQueryRequest<EspQuery>): Observable<DataQueryResponse> {
    options.liveStreaming = true;

    const queryResponses = super.query(options);

    return queryResponses.pipe(
      map((event: DataQueryResponse) => {
        const grafanaError = DataSource.getLastGrafanaError(event);
        if (grafanaError || !event.data.length) {
          return event;
        }

        const lastFrame = event.data.at(-1);

        const lastPluginError = DataSource.getLastPluginError(event);
        if (lastPluginError) {
            DataSource.addErrorToEvent(event, lastPluginError);
            event.data.splice(0, event.data.length);
            return event;
        }

        const opcodeField = lastFrame.fields.find((field: any) => field.name === '@opcode');
        if (opcodeField) {
          const opcodeValues = DataSource.getFieldValues(opcodeField);

          const lastErrorIndex = opcodeValues.lastIndexOf('error');
          const lastErrorClearIndex = opcodeValues.lastIndexOf('error-clear');
          const indexToStartFrom = Math.max(lastErrorIndex, lastErrorClearIndex);
          if (indexToStartFrom > -1) {
            for (const field of lastFrame.fields) {
              const isLegacyField = !(field.values instanceof Array);
              if (isLegacyField) {
                  // We are dealing with an out-of-date (9.x) version of Grafana which is using a deprecated values type.
                  field.values.buffer.splice(0, indexToStartFrom + 1);
                  continue;
              }

              field.values.splice(0, indexToStartFrom + 1);
            }
          }
        }

        return event;
      })
    );
  }

    private static getLastGrafanaError(event: DataQueryResponse): DataQueryError | undefined {
        if (event.error) {
            // We are dealing with an out-of-date (9.x) version of Grafana which is using the deprecated error field.
            return event.error;
        }

        return event.errors?.at(-1);
    }

    private static getLastPluginError(event: any): DataQueryError | undefined {
        const lastFrame = event.data.at(-1);
        const errorField = lastFrame.fields.find((field: any) => field.name === '@error');
        if (!errorField || !errorField.values.length) {
            return;
        }

        const lastValue = DataSource.getLastPluginErrorFieldValue(errorField);
        if (lastValue == null) {
            return;
        }

        return {message: lastValue};
    }

    private static getLastPluginErrorFieldValue(pluginErrorField: any): any | undefined {
        const isLegacyField = !(pluginErrorField.values instanceof Array);
        if (isLegacyField) {
            // We are dealing with an out-of-date (9.x) version of Grafana which is using a deprecated values type.
            return pluginErrorField.values.buffer.lastItem;
        }

        return pluginErrorField.values.at(-1);
    }

    private static addErrorToEvent(event: any, error: DataQueryError): void {
        event.error = error;
        event.errors = event.errors ? event.errors.concat([error]) : [error];
    }

    private static getFieldValues(field: any): any[] {
        const isLegacyField = !(field.values instanceof Array);
        if (isLegacyField) {
            // We are dealing with an out-of-date (9.x) version of Grafana which is using a deprecated values type.
            return field.values.buffer;
        }

        return field.values;
    }
}
