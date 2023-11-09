/*
	Copyright © 2023, SAS Institute Inc., Cary, NC, USA.  All Rights Reserved.
	SPDX-License-Identifier: Apache-2.0
*/

import React, {useEffect, useMemo, useRef} from 'react';
import {Checkbox, HorizontalGroup, InlineLabel, Select, VerticalGroup} from '@grafana/ui';
import {DataSourcePluginOptionsEditorProps, SelectableValue} from '@grafana/data';
import {EspDataSourceOptions} from '../types';

ConfigEditor.DISCOVERY_DEFAULT_OPTIONS = [
    {label: 'SAS Event Stream Manager', value: 'http://sas-event-stream-manager-app/SASEventStreamManager'},
    {label: 'SAS Event Stream Processing Studio', value: 'http://sas-event-stream-processing-studio-app/SASEventStreamProcessingStudio'},
];

ConfigEditor.DISCOVERY_DEFAULT_VIYA_OPTIONS = [
    {label: 'SAS Event Stream Manager', value: 'https://sas-event-stream-manager-app/SASEventStreamManager'},
    {label: 'SAS Event Stream Processing Studio', value: 'https://sas-event-stream-processing-studio-app/SASEventStreamProcessingStudio'},
];

export function ConfigEditor({options, onOptionsChange}: DataSourcePluginOptionsEditorProps<EspDataSourceOptions>) {
    const {jsonData} = options;

    const getDiscoveryOptions = (isViya: boolean) => isViya ? ConfigEditor.DISCOVERY_DEFAULT_VIYA_OPTIONS : ConfigEditor.DISCOVERY_DEFAULT_OPTIONS;

    const deriveSelectedOptionFromUrl = (discoveryServiceUrl: string | null, discoveryOptions: Array<SelectableValue<string>>) => {
        if (!discoveryServiceUrl) {
            return null;
        }

        const matchingOption = discoveryOptions.find(option => option.value === discoveryServiceUrl);
        return matchingOption ?? {value: discoveryServiceUrl, label: discoveryServiceUrl};
    }

    const changePropOptions = (change: Object) => {
        const newOptions = {...options, ...change};
        onOptionsChange(newOptions);
    }

    const changePropOptionsJsonData = (change: Object) => {
        const newJsonData = {...jsonData, ...change};
        changePropOptions({jsonData: newJsonData});
    }

    const handleDiscoveryServiceProviderChange = (selectedOption: SelectableValue<string>) => {
        changePropOptions({url: selectedOption.value});
    }

    const handleTlsSkipVerifyCheckboxChange = (checked: boolean) => {
        changePropOptionsJsonData({tlsSkipVerify: checked});
    }

    const handleViyaCheckboxChange = (checked: boolean) => {
        const isViya = checked;
        // Grafana will ignore attempts to reset datasource URLs and will revert to the previously saved value upon a future save, rather than persist a falsy URL.
        // To prevent this unwanted behaviour, a default URL is chosen to override the existing URL if possible.
        const defaultUrl = getDiscoveryOptions(isViya)?.at(0)?.value;
        changePropOptions({
            url: defaultUrl ?? "",
            jsonData: {...jsonData, isViya: isViya}
        });
    }

    const mountEffectRefIsViya = useRef(jsonData.isViya);
    const mountEffectRefChangePropOptionsJsonData = useRef(changePropOptionsJsonData);
    useEffect(() => {
        (async () => {
            const isViya = mountEffectRefIsViya.current;
            const changePropOptionsJsonData = mountEffectRefChangePropOptionsJsonData.current;

            let jsonDataChanges = new Map<string, Object>();
            jsonDataChanges.set("oauthPassThru", true);

            if (isViya == null) {
                let isViya: boolean = await fetch(`${window.location.origin}/SASLogon/`).then((response) => response.ok, () => false);
                jsonDataChanges.set("isViya", isViya);
            }

            const jsonDataDelta = Object.fromEntries(jsonDataChanges);
            changePropOptionsJsonData(jsonDataDelta);
        })()
    }, []);

    const discoveryOptions = getDiscoveryOptions(jsonData.isViya);
    const selectedDiscoveryOption = useMemo(() => deriveSelectedOptionFromUrl(options.url, discoveryOptions), [options.url, discoveryOptions]);

    return (
        <VerticalGroup>
            <HorizontalGroup>
                <InlineLabel width="auto">Discovery service provider</InlineLabel>
                <Checkbox label="Viya" value={jsonData.isViya ?? false} onChange={e => handleViyaCheckboxChange(e.currentTarget.checked)}/>
                <Select key={`${jsonData.isViya}`}
                        options={discoveryOptions} value={selectedDiscoveryOption}
                        allowCustomValue onCreateOption={customValue => handleDiscoveryServiceProviderChange({value: customValue, label: customValue})}
                        onChange={handleDiscoveryServiceProviderChange}
                />
            </HorizontalGroup>
            <Checkbox label="(Insecure) Skip TLS certificate verification" value={jsonData.tlsSkipVerify ?? false}
                      onChange={e => handleTlsSkipVerifyCheckboxChange(e.currentTarget.checked)}
            />
        </VerticalGroup>
    );
}
