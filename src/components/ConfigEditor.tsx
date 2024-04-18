/*
	Copyright © 2023, SAS Institute Inc., Cary, NC, USA.  All Rights Reserved.
	SPDX-License-Identifier: Apache-2.0
*/

import React, {useMemo, useState} from 'react';
import {Checkbox, Field, InlineLabel, Input, Select, Stack} from '@grafana/ui';
import {DataSourcePluginOptionsEditorProps, SelectableValue} from '@grafana/data';
import {EspDataSourceOptions} from '../types';

interface DiscoveryOption {
    label: string,
    value: string
}

ConfigEditor.DISCOVERY_DEFAULT_OPTIONS_NO_TLS = [
    {label: 'SAS Event Stream Manager', value: 'http://sas-event-stream-manager-app/SASEventStreamManager'},
    {label: 'SAS Event Stream Processing Studio', value: 'http://sas-event-stream-processing-studio-app/SASEventStreamProcessingStudio'},
];

ConfigEditor.DISCOVERY_DEFAULT_OPTIONS_TLS = [
    {label: 'SAS Event Stream Manager', value: 'https://sas-event-stream-manager-app/SASEventStreamManager'},
    {label: 'SAS Event Stream Processing Studio', value: 'https://sas-event-stream-processing-studio-app/SASEventStreamProcessingStudio'},
];

enum DISCOVERY_TYPE_OPTION_VALUES {DEFAULT, URL}
ConfigEditor.DISCOVERY_TYPE_OPTIONS = [
    {label: "Internal", value: DISCOVERY_TYPE_OPTION_VALUES.DEFAULT},
    {label: "URL", value: DISCOVERY_TYPE_OPTION_VALUES.URL}
];

ConfigEditor.stringToUrl = (urlString: string) => {
    let url;
    try {
        url = new URL(urlString);
    } catch (e: unknown) {
        url = null;
    }

    return url;
}

ConfigEditor.getDiscoveryOptions = (tls: boolean) => tls ? ConfigEditor.DISCOVERY_DEFAULT_OPTIONS_TLS : ConfigEditor.DISCOVERY_DEFAULT_OPTIONS_NO_TLS;

ConfigEditor.deriveSelectedOptionFromUrl = (discoveryServiceUrlString: string) => {
    const discoveryServiceUrl = ConfigEditor.stringToUrl(discoveryServiceUrlString);
    const isDiscoveryServiceTlsEnabled = discoveryServiceUrl ? ConfigEditor.isDiscoveryServiceUrlTls(discoveryServiceUrl) : true;
    const discoveryOptions = ConfigEditor.getDiscoveryOptions(isDiscoveryServiceTlsEnabled);

    return discoveryOptions.find(option => option.value === discoveryServiceUrlString);
}

ConfigEditor.deriveSelectedDiscoveryTypeFromUrl = (discoveryServiceUrlString: string) => {
    if (!discoveryServiceUrlString) {
        return DISCOVERY_TYPE_OPTION_VALUES.DEFAULT;
    }

    const discoveryServiceUrl = ConfigEditor.stringToUrl(discoveryServiceUrlString);
    if (!discoveryServiceUrl) {
        return DISCOVERY_TYPE_OPTION_VALUES.URL;
    }

    const isDiscoveryServiceTlsEnabled = ConfigEditor.isDiscoveryServiceUrlTls(discoveryServiceUrl);
    const defaultDiscoveryOptions = ConfigEditor.getDiscoveryOptions(isDiscoveryServiceTlsEnabled);
    const defaultUrlMatched = defaultDiscoveryOptions.some(option => option.value === discoveryServiceUrlString);

    return defaultUrlMatched ? DISCOVERY_TYPE_OPTION_VALUES.DEFAULT : DISCOVERY_TYPE_OPTION_VALUES.URL;
}

ConfigEditor.isDiscoveryServiceUrlTls = (discoveryServiceUrl: URL) => {
    return discoveryServiceUrl.protocol === "https:";
}

export function ConfigEditor({options, onOptionsChange}: Readonly<DataSourcePluginOptionsEditorProps<EspDataSourceOptions>>) {
    const {jsonData} = options;

    const [selectedDiscoveryType, setSelectedDiscoveryType] = useState(() => ConfigEditor.deriveSelectedDiscoveryTypeFromUrl(options.url));

    const changePropOptions = (change: Object) => {
        const newOptions = {...options, ...change};
        onOptionsChange(newOptions);
    }

    const changePropOptionsJsonData = (change: Object) => {
        const newJsonData = {...jsonData, ...change};
        changePropOptions({jsonData: newJsonData});
    }

    const handleDiscoveryServiceUrlChange = (newUrl: string) => {
        changePropOptions({url: newUrl});
    }

    const handleTlsSkipVerifyCheckboxChange = (checked: boolean) => {
        changePropOptionsJsonData({tlsSkipVerify: checked});
    }

    const handleDiscoveryServiceTypeChange = (selectable: SelectableValue<DISCOVERY_TYPE_OPTION_VALUES>) => {
        setSelectedDiscoveryType(selectable?.value ?? DISCOVERY_TYPE_OPTION_VALUES.DEFAULT);
    }

    const handleOauthPassthroughCheckboxChange = (checked: boolean) => {
        changePropOptionsJsonData({oauthPassThru: checked});
    }

    const handleTlsCheckboxChange = (checked: boolean) => {
        const discoveryServiceUrl = ConfigEditor.stringToUrl(options.url);
        if (!discoveryServiceUrl) {
            return;
        }

        const isUrlHttps = ConfigEditor.isDiscoveryServiceUrlTls(discoveryServiceUrl)
        if (isUrlHttps !== checked) {
            discoveryServiceUrl.protocol = checked ? "https:" : "http:";
            changePropOptions({url: discoveryServiceUrl.toString()});
        }
    }

    const discoveryServiceUrl = ConfigEditor.stringToUrl(options.url);
    const isDiscoveryServiceTlsEnabled = discoveryServiceUrl ? ConfigEditor.isDiscoveryServiceUrlTls(discoveryServiceUrl) : true;
    const discoveryOptions = ConfigEditor.getDiscoveryOptions(isDiscoveryServiceTlsEnabled);
    const selectedDiscoveryOption = useMemo(() => ConfigEditor.deriveSelectedOptionFromUrl(options.url), [options.url]);

    return (
        <Stack direction="column" alignItems="start">
            <div style={{["margin-bottom" as string]: "10px"}}>
                <Checkbox label="Do not use TLS  certificate validation (not recommended)" value={jsonData.tlsSkipVerify ?? false}
                          onChange={e => handleTlsSkipVerifyCheckboxChange(e.currentTarget.checked)}
                />
            </div>
            <div style={{["display" as string]: "grid", ["grid-template" as string]: "'labels fields' / 1fr auto"}}>
                <InlineLabel width="auto">Discovery service</InlineLabel>
                <Stack direction="column" alignItems="start">
                    <Select options={ConfigEditor.DISCOVERY_TYPE_OPTIONS} value={selectedDiscoveryType} onChange={handleDiscoveryServiceTypeChange}/>
                    <DiscoveryTypeForm type={selectedDiscoveryType}
                                       discoveryUrLOptions={discoveryOptions} selectedDiscoveryUrlOption={selectedDiscoveryOption}
                                       url={options.url} onUrlChange={handleDiscoveryServiceUrlChange}
                                       oauth={jsonData.oauthPassThru} onOauthChange={handleOauthPassthroughCheckboxChange}
                                       tls={isDiscoveryServiceTlsEnabled} onTlsChange={handleTlsCheckboxChange}/>
                </Stack>
            </div>
        </Stack>
    );
}

function DiscoveryTypeForm(props: Readonly<{ type: DISCOVERY_TYPE_OPTION_VALUES
                                             discoveryUrLOptions: DiscoveryOption[], selectedDiscoveryUrlOption: DiscoveryOption | undefined,
                                             url: string, onUrlChange: Function
                                             oauth: boolean | undefined, onOauthChange: Function,
                                             tls: boolean | undefined, onTlsChange: Function,
                                           }>) {
    return (<>{props.type === DISCOVERY_TYPE_OPTION_VALUES.DEFAULT ?
        <DiscoveryFormDefault discoveryUrLOptions={props.discoveryUrLOptions} selectedDiscoveryUrlOption={props.selectedDiscoveryUrlOption} onUrlChange={props.onUrlChange}
                              oauth={props.oauth} onOauthChange={props.onOauthChange}
                              tls={props.tls} onTlsChange={props.onTlsChange}/> :
        <DiscoveryFormUrl oauth={props.oauth} onOauthChange={props.onOauthChange}
                          url={props.url} onUrlChange={props.onUrlChange}/>}
    </>);
}

function DiscoveryFormDefault(props: Readonly<{ oauth: boolean | undefined, onOauthChange: Function,
                                                discoveryUrLOptions: DiscoveryOption[], selectedDiscoveryUrlOption: DiscoveryOption | undefined, onUrlChange: Function,
                                                tls: boolean | undefined, onTlsChange: Function
                                              }>) {
    return (<>
        <Select options={props.discoveryUrLOptions} value={props.selectedDiscoveryUrlOption} onChange={selectable => props.onUrlChange(selectable.value ?? "")}/>
        <Stack>
            <OauthCheckbox value={props.oauth} onChange={props.onOauthChange}/>
            <Checkbox label="TLS" value={props.tls} onChange={e => props.onTlsChange(e.currentTarget.checked)}/>
        </Stack>
    </>);
}

function DiscoveryFormUrl(props: Readonly<{ oauth: boolean | undefined, onOauthChange: Function, url: string, onUrlChange: Function}>) {
    return (<>
        <Field>
            <Input placeholder={"Enter a URL"} width={80} value={props.url} onChange={e => props.onUrlChange(e.currentTarget.value)}/>
        </Field>
        <OauthCheckbox value={props.oauth} onChange={props.onOauthChange}/>
    </>);
}

function OauthCheckbox(props: Readonly<{value: boolean | undefined, onChange: Function}>) {
    return (<Checkbox label="OAuth token" value={props.value ?? false} onChange={e => props.onChange(e.currentTarget.checked)}/>);
}
