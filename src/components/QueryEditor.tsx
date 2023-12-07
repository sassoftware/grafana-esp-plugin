/*
	Copyright © 2023, SAS Institute Inc., Cary, NC, USA.  All Rights Reserved.
	SPDX-License-Identifier: Apache-2.0
*/

import React, {PureComponent} from 'react';
import {Select, SelectCommonProps} from '@grafana/ui';
import {QueryEditorProps, SelectableValue} from '@grafana/data';
import {DataSource} from '../datasource';
import {
  ContinuousQuery,
  EspDataSourceOptions,
  EspObject,
  EspObjectType,
  EspQuery,
  Field,
  getEspObjectType,
  Project,
  Server,
  Window,
} from '../types';

type Props = QueryEditorProps<DataSource, EspQuery, EspDataSourceOptions>;

interface ServersResponse {
  data: Server[]
}

interface State {
  isServerDataFetched: boolean;
  serverOptions: Array<SelectableEspObject<Server>>;
  projectOptions: Array<SelectableEspObject<Project>>;
  cqOptions: Array<SelectableEspObject<ContinuousQuery>>;
  windowOptions: Array<SelectableEspObject<Window>>;
  fieldOptions: Array<SelectableEspObject<Field>>;
  selectedServer: Server | null | undefined;
  selectedProject: Project | null | undefined;
  selectedCq: ContinuousQuery | null | undefined;
  selectedWindow: Window | null | undefined;
  selectedFields: Field[];
}

export class QueryEditor extends PureComponent<Props> {
  state: State;
  espQueryController: EspQueryController;

  constructor(props: Props) {
    super(props);

    this.state = {
      isServerDataFetched: false,
      serverOptions: [],
      projectOptions: [],
      cqOptions: [],
      windowOptions: [],
      fieldOptions: [],
      selectedServer: undefined,
      selectedProject: undefined,
      selectedCq: undefined,
      selectedWindow: undefined,
      selectedFields: []
    };

    this.espQueryController = new EspQueryController(this.props.query, this.props.onChange, this.props.onRunQuery);
  }

  componentDidMount() {
    this.props.datasource
      .getResource('servers')
      .then(async (serversResponse: ServersResponse) => {
        const servers = serversResponse.data
        const [server, project, cq, window, fields] = QueryEditor.deriveSelectionsFromQuery(servers, this.props.query);
        await this.setStateWithPromise({
          selectedServer: server,
          selectedProject: project,
          selectedCq: cq,
          selectedWindow: window,
          selectedFields: fields
        });
        await this.updateServers(servers);
        await this.setStateWithPromise({ isServerDataFetched: true });
      })
      .catch(console.error);
  }

  render() {
    const state = this.state;

    if (!state.isServerDataFetched) {
      return <span>Discovering ESP servers...</span>;
    }

    function findSelected(selectables: Array<SelectableEspObject<EspObject>>, queryEspObject: EspObject | null | undefined) {
      if (!queryEspObject) {
        return null;
      }

      return selectables.find((selectable) => selectable.value.name === queryEspObject.name) ?? null;
    }

    function findMultiSelected(selectables: Array<SelectableEspObject<EspObject>>, queryEspObject: EspObject[]): Array<SelectableEspObject<EspObject>> {
      const selectedEspObjectNames: Set<string> = new Set(queryEspObject.map(eo => eo.name));
      return selectables.filter((selectable) => selectedEspObjectNames.has(selectable.value.name));
    }

    const selectedServerOption = findSelected(state.serverOptions, state.selectedServer);
    const selectedProjectOption = findSelected(state.projectOptions, state.selectedProject);
    const selectedCqOption = findSelected(state.cqOptions, state.selectedCq);
    const selectedWindowOption = findSelected(state.windowOptions, state.selectedWindow);
    const selectedFieldOptions = findMultiSelected(state.fieldOptions, state.selectedFields);

    const selectArgs: Array<Partial<SelectCommonProps<EspObject>>> = [
      { id: 'server', options: state.serverOptions, value: selectedServerOption, placeholder: 'ESP server' },
      { id: 'project', options: state.projectOptions, value: selectedProjectOption, placeholder: 'ESP project' },
      { id: 'cq', options: state.cqOptions, value: selectedCqOption, placeholder: 'Continuous query' },
      { id: 'window', options: state.windowOptions, value: selectedWindowOption, placeholder: 'Window' },
    ];

    return (
      <div>
        {selectArgs.map((arg) => (
          <Select
            key={arg.id}
            isMulti={false}
            isClearable={false}
            backspaceRemovesValue={false}
            options={arg.options}
            onChange={this.onSelect}
            value={arg.value}
            isSearchable={true}
            maxMenuHeight={500}
            placeholder={arg.placeholder}
            noOptionsMessage={'No options found'}
          />
        ))}
        <Select
            key={'fields'}
            isMulti={true}
            isClearable={true}
            backspaceRemovesValue={true}
            options={state.fieldOptions}
            onChange={v => this.onMultiSelect(v, EspObjectType.FIELD)}
            value={selectedFieldOptions}
            isSearchable={true}
            maxMenuHeight={500}
            placeholder={'Fields'}
            noOptionsMessage={'No options found'}
        />
      </div>
    );
  }

  onSelect = async (selectableValue: SelectableValue<EspObject>) => {
    if (!selectableValue.value) {
      throw Error('Expected selection event to provide a selectable value.');
    }

    const espObject: EspObject = selectableValue.value;

    const type = getEspObjectType(espObject);

    this.espQueryController.set(espObject);

    switch (type) {
      case EspObjectType.SERVER:
        await this.setSelectedServer(espObject as Server);
        await this.updateProjects((espObject as Server).projects);
        break;
      case EspObjectType.PROJECT:
        await this.setSelectedProject(espObject as Project);
        await this.updateContinuousQueries((espObject as Project).continuousQueries);
        break;
      case EspObjectType.CONTINUOUS_QUERY:
        await this.setSelectedCq(espObject as ContinuousQuery);
        await this.updateWindows((espObject as ContinuousQuery).windows);
        break;
      case EspObjectType.WINDOW:
        await this.setSelectedWindow(espObject as Window);
        await this.updateFields((espObject as Window).fields)
        break;
    }

    await this.clearSelectedSubFields(type);
  };

  private async clearSelectedSubFields(type: EspObjectType) {
    switch (type) {
      case EspObjectType.SERVER:
        await this.setSelectedProject(null);
      case EspObjectType.PROJECT:
        await this.setSelectedCq(null);
      case EspObjectType.CONTINUOUS_QUERY:
        await this.setSelectedWindow(null);
      case EspObjectType.WINDOW:
        await this.setSelectedFields(null);
      case EspObjectType.FIELD:
        break;
    }
  }

  onMultiSelect = async (selectableValues: SelectableValue<EspObject>, selectableType: EspObjectType) => {
    if (!(selectableValues instanceof Array)) {
      throw new Error('Expected multi-select event to provide a collection of values.');
    }

    const potentialFields: EspObject[] = [];
    for (const selectableValue of selectableValues) {
      if (selectableValue.value == null) {
        throw Error('Unexpected null value in multi-selection event.');
      }

      potentialFields.push(selectableValue.value);
    }

    switch (selectableType) {
      case EspObjectType.FIELD:
        const fields = potentialFields as Field[];
        this.espQueryController.setFields(fields);
        await this.setSelectedFields(fields);
        break;
      default:
        throw new Error('Unexpected value type in multi-selection event.')
    }
  };

  static deriveSelectionsFromQuery(servers: Server[], query: EspQuery): [Server?, Project?, ContinuousQuery?, Window?, Field[]?] {
    let selectedServer: Server | undefined;
    let selectedProject: Project | undefined;
    let selectedCq: ContinuousQuery | undefined;
    let selectedWindow: Window | undefined;
    let selectedFields: Field[] | undefined = [];
    const returnValue = (): [Server?, Project?, ContinuousQuery?, Window?, Field[]?] => [selectedServer, selectedProject, selectedCq, selectedWindow, selectedFields];

    selectedServer = servers.find((server: Server) => server.externalUrl === query.externalServerUrl) ?? undefined;
    if (!selectedServer) {
      return returnValue();
    }

    const projects = selectedServer?.projects ?? [];
    selectedProject = projects.find((project: Project) => project.name === query.projectName) ?? undefined;
    if (!selectedProject) {
      return returnValue();
    }

    const cqs = selectedProject?.continuousQueries ?? [];
    selectedCq = cqs.find((cq: ContinuousQuery) => cq.name === query.cqName) ?? undefined;
    if (!selectedCq) {
      return returnValue();
    }

    const windows = selectedCq?.windows ?? [];
    selectedWindow = windows.find((window: Window) => window.name === query.windowName) ?? undefined;
    if (!selectedWindow) {
      return returnValue();
    }

    const fields = selectedWindow?.fields ?? [];
    selectedFields = fields.filter((field: Field) => query.fields.includes(field.name)) ?? [];

    return returnValue();
  }

  async updateServers(servers: Server[]) {
    const serverSelectables: Array<SelectableImpl<Server>> = servers.map(SelectableImpl.fromObject);
    await this.setStateWithPromise({ serverOptions: serverSelectables });

    const selectedServer = this.state.selectedServer;
    if (selectedServer) {
      await this.updateProjects(selectedServer.projects);
    }
  }

  async updateProjects(projects: Project[]) {
    const projectSelectables: Array<SelectableImpl<Project>> = projects.map(SelectableImpl.fromObject);
    await this.setStateWithPromise({ projectOptions: projectSelectables });

    const selectedProject = this.state.selectedProject;
    if (selectedProject) {
      await this.updateContinuousQueries(selectedProject.continuousQueries);
    }
  }

  async updateContinuousQueries(continuousQueries: ContinuousQuery[]) {
    const continuousQuerySelectables: Array<SelectableImpl<ContinuousQuery>> = continuousQueries.map(
      SelectableImpl.fromObject
    );
    await this.setStateWithPromise({ cqOptions: continuousQuerySelectables });

    const selectedCq = this.state.selectedCq;
    if (selectedCq) {
      await this.updateWindows(selectedCq.windows);
    }
  }

  async updateWindows(windows: Window[]) {
    const windowSelectables: Array<SelectableImpl<Window>> = windows.map(SelectableImpl.fromObject);
    await this.setStateWithPromise({ windowOptions: windowSelectables });

    const selectedWindow = this.state.selectedWindow;
    if (selectedWindow) {
      await this.updateFields(selectedWindow.fields);
    }
  }

  async updateFields(fields: Field[]) {
    const fieldSelectables: Array<SelectableImpl<Field>> = fields.map(SelectableImpl.fromObject);
    await this.setStateWithPromise({fieldOptions: fieldSelectables});
  }

  async setSelectedServer(server: Server | null) {
    await this.setStateWithPromise({ selectedServer: server });
  }

  async setSelectedProject(project: Project | null): Promise<void> {
    await this.setStateWithPromise({ selectedProject: project });
  }

  async setSelectedCq(cq: ContinuousQuery | null) {
    await this.setStateWithPromise({ selectedCq: cq });
  }

  async setSelectedWindow(window: Window | null) {
    await this.setStateWithPromise({ selectedWindow: window });

    if (window != null) {
      this.espQueryController.save();
      this.espQueryController.execute();
    }
  }

  async setSelectedFields(fields: Field[] | null) {
    await this.setStateWithPromise({selectedFields: fields ?? []})

    if (fields != null) {
      this.espQueryController.save();
      this.espQueryController.execute();
    }
  }

  private setStateWithPromise(stateDiff: {}): Promise<void> {
    return new Promise(resolve => this.setState(stateDiff, resolve))
  }

}

interface SelectableEspObject<EspObject> extends SelectableValue<EspObject> {
  value: EspObject;
  label: string;
  title: string;
}

class SelectableImpl<T> implements SelectableEspObject<EspObject> {
  value: EspObject;
  label: string;
  title: string;

  constructor(selectableObject: EspObject) {
    this.value = selectableObject;
    this.label = selectableObject.name;
    this.title = this.label;
  }

  static fromObject<T>(object: EspObject) {
    return new SelectableImpl<T>(object);
  }
}

class EspQueryController {
  espQuery: EspQuery;
  onChange: (espQuery: EspQuery) => void;
  onRunQuery: () => void;

  constructor(espQuery: EspQuery, onChange: (espQuery: EspQuery) => void, onRunQuery: () => void) {
    this.espQuery = espQuery;
    this.onChange = onChange;
    this.onRunQuery = onRunQuery;
  }

  save(): void {
    this.onChange(this.espQuery);
  }

  execute(): void {
    this.onRunQuery();
  }

  set(espObject: EspObject): void {
    const type: EspObjectType = getEspObjectType(espObject);

    this.clearQuerySubFields(type);

    switch (type) {
      case EspObjectType.SERVER:
        this.setServer(espObject as Server);
        break;
      case EspObjectType.PROJECT:
        this.setProject(espObject as Project);
        break;
      case EspObjectType.CONTINUOUS_QUERY:
        this.setContinuousQuery(espObject as ContinuousQuery);
        break;
      case EspObjectType.WINDOW:
        this.setWindow(espObject as Window);
        break;
    }
  }

  private clearQuerySubFields(espObjectType: EspObjectType): void {
    switch (espObjectType) {
      case EspObjectType.SERVER:
        this.setProject(null);
      case EspObjectType.PROJECT:
        this.setContinuousQuery(null);
      case EspObjectType.CONTINUOUS_QUERY:
        this.setWindow(null);
      case EspObjectType.WINDOW:
        this.setFields([]);
      case EspObjectType.FIELD:
        return;
    }
  }

  private setServer(server: Server | null): void {
    this.espQuery.externalServerUrl = server ? server.externalUrl : null;
    this.espQuery.internalServerUrl = server ? server.url : null;
  }

  private setProject(project: Project | null): void {
    this.espQuery.projectName = project?.name ?? null;
  }

  private setContinuousQuery(continuousQuery: ContinuousQuery | null): void {
    this.espQuery.cqName = continuousQuery?.name ?? null;
  }

  private setWindow(window: Window | null): void {
    this.espQuery.windowName = window?.name ?? null;
  }

  setFields(fields: Field[]): void {
    this.espQuery.fields = fields.map(field => field.name);
  }
}
