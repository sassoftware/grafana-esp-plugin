/*
	Copyright © 2023, SAS Institute Inc., Cary, NC, USA.  All Rights Reserved.
	SPDX-License-Identifier: Apache-2.0
*/

import React, {PureComponent} from 'react';
import {InlineLabel, Select} from '@grafana/ui';
import {DataSourcePluginOptionsEditorProps, SelectableValue} from '@grafana/data';
import { EspDataSourceOptions } from '../types';

interface Props extends DataSourcePluginOptionsEditorProps<EspDataSourceOptions> {}

interface State {
  selectedOption: SelectableValue<string> | undefined;
  customOptions: Array<SelectableValue<string>>;
}

export class ConfigEditor extends PureComponent<Props> {
  private static DISCOVERY_URL_DEFAULT_STUDIO = 'http://sas-event-stream-processing-studio-app/SASEventStreamProcessingStudio';
  private static DISCOVERY_URL_DEFAULT_ESM = 'http://sas-event-stream-manager-app/SASEventStreamManager';
  private static DISCOVERY_DEFAULT_OPTIONS = [
    {label: 'SAS Event Stream Manager', value: ConfigEditor.DISCOVERY_URL_DEFAULT_ESM},
    {label: 'SAS Event Stream Processing Studio', value: ConfigEditor.DISCOVERY_URL_DEFAULT_STUDIO},
  ];

  state: State;

  constructor(props: Props) {
    super(props);

    this.props.options.jsonData.oauthPassThru = true;

    this.state = {
      selectedOption: undefined,
      customOptions: [],
    };

    let discoveryServiceUrl = this.getDiscoveryServiceUrlFromProps();
    if (!discoveryServiceUrl) {
      this.state.selectedOption = ConfigEditor.DISCOVERY_DEFAULT_OPTIONS[0];
    } else {
      const matchingOption = ConfigEditor.DISCOVERY_DEFAULT_OPTIONS.find(option => option.value === discoveryServiceUrl);
      if (matchingOption) {
        this.state.selectedOption = matchingOption;
      } else {
        const customOption: SelectableValue<string> = {value: discoveryServiceUrl, label: discoveryServiceUrl};
        this.state.customOptions = [...this.state.customOptions, customOption];
        this.state.selectedOption = customOption;
      }
    }
    this.setDiscoveryServiceUrlInProps(this.state.selectedOption.value ?? "");

    this.handleDiscoveryServiceProviderChange = this.handleDiscoveryServiceProviderChange.bind(this);
  }

  private handleDiscoveryServiceProviderChange(selectedOption: SelectableValue<string>) {
    this.setState({selectedOption: selectedOption});
    this.setDiscoveryServiceUrlInProps(selectedOption.value ?? "");
  }

  private setDiscoveryServiceUrlInProps(newUrl: string) {
    this.props.options.url = newUrl;
    this.props.onOptionsChange(this.props.options);
  }

  private getDiscoveryServiceUrlFromProps(): string | undefined {
    return this.props.options.url;
  }

  render() {
    return (
      <div className="gf-form-group">
        <div className="gf-form">
          <InlineLabel width="auto">
            Discovery service provider
          </InlineLabel>
          <Select
              options={[...ConfigEditor.DISCOVERY_DEFAULT_OPTIONS, ...this.state.customOptions]}
              value={this.state.selectedOption}
              allowCustomValue
              onCreateOption={customValue => {
                const customOption: SelectableValue<string> = {value: customValue, label: customValue};
                this.setState( (prevState: State) => {
                  return {
                    customOptions: [...prevState.customOptions, customOption],
                  };
                });
                this.handleDiscoveryServiceProviderChange(customOption);
              }}
              onChange={this.handleDiscoveryServiceProviderChange}
          />
        </div>
      </div>
    );
  }
}
