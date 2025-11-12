# SAS Event Stream Processing Data Source Plug-in for Grafana

## Overview
The SAS Event Stream Processing Data Source Plug-in for Grafana enables you to discover and stream data from ESP servers in SAS Event Stream Processing in a Kubernetes environment. 

The plug-in is intended for visualizing event streams and provides an alternative to using SAS Event Stream Streamviewer. The plug-in is not intended to be used as a monitoring tool.

Here is an example of a Grafana dashboard for an ESP project. This dashboard relates to the Sailing example that is discussed in [Examples](#examples).

<img alt="Sailing dashboard" src="src/img/sailing-dashboard.png"  width="50%" height="50%">


## Getting Started
The following steps provide an example of how to get started with the plug-in. 

### Check Prerequisites
* A running deployment of SAS Event Stream Processing in a Kubernetes environment such as the Microsoft Marketplace or SAS Viya platform.
* A Grafana instance that is configured to use the same OAuth provider as SAS Event Stream Processing, typically SAS Logon.
* The plug-in is automatically deployed with the SAS Event Stream Processing app in the Microsoft Marketplace. When SAS Event Stream Processing is deployed with the SAS Viya platform, you can deploy the plug-in by completing the instructions in [Deploying the Plug-in](#deploying-the-plug-in).
* An ESP project that can be run in either SAS Event Stream Processing Studio, SAS Event Stream Manager, or ESP server.

To visualize data, you must have an ESP project running in either SAS Event Stream Processing Studio, SAS Event Stream Manager, or ESP server.

### Add the SAS Event Stream Processing Data Source
1. In the **Data Sources** section find and select **SAS Event Stream Processing Data Source**.
2. (Optional) If you are using a self-signed certificate, you can select the **Do not use TLS certificate validation (not recommended)** check box. This option is not suitable for production use.
3. Choose how to connect to your ESP server:
   - **Discovery Service**: A discovery service is provided by SAS Event Stream Processing Studio and SAS Event Stream Manager and is used to connect to ESP servers. Use the **Host type** drop-down menu to select one of the following options:
     - **Internal Discovery Service**: This option assumes that Grafana has been deployed in the same namespace as SAS Event Stream Processing Studio and SAS Event Stream Manager.
       Follow the instructions in [Deploy a Released Version of the Plug-in](#deploy-a-released-version-of-the-plug-in).
     - **Discovery Service URL**: Use this option to specify the location of the discovery service.
   - **Direct ESP Server URL**: Use this option to connect directly to a specific ESP server instance by providing its URL and connection details. This option bypasses the
     discovery service and is useful if you know the exact ESP server endpoint you want to use.
4. If you selected **Internal Discovery Service** in the previous step, another drop-down menu is displayed. Select either **SAS Event Stream Manager** or **SAS Event Stream Processing Studio** as the discovery service, depending on where you prefer to run ESP projects.
5. By default, the **TLS** check box is selected. If the data source does not use TLS, clear this check box.
6. Select the **OAuth token** check box if OAuth tokens are used by the discovery service and you want to forward the token to the discovery service and ESP servers.
7. Click **Save & test**.</br>The plug-in attempts to connect to your chosen discovery service.
8. (Optional) Repeat [steps 1-4](#add-the-sas-event-stream-processing-data-source) to add another data source. For example, if you added SAS Event Stream Manager as a data source, you can repeat the steps to add SAS Event Stream Processing Studio as an additional data source if needed.

### Connect a Panel to SAS Event Stream Processing as a Data Source
1. Create a new dashboard and add a panel.
2. In the **Query** tab at the bottom of the panel editor, select the data source that you configured previously.</br>The plug-in discovers running instances of ESP servers by connecting to your chosen data source. When the connection is successful, the **Query** tab shows drop-down menus that are related to SAS Event Stream Processing.
3. From the **ESP server** drop-down menu, select the ESP server that you want to query. You can filter the available options by entering a keyword and then selecting the desired ESP server from the drop-down menu.
4. From the **ESP project**, **Continuous query**, and **Window** drop-down menus, select appropriate values until you are able to narrow the query down to the desired target window in the ESP project.</br>When an available target window is selected, the plug-in establishes a connection and starts querying for new events.
5. From the **Fields** drop-down menu, select the fields (from the window in your ESP project) that you want to visualize.
6. If required, change the visualization type from the default of **Time series** to a visualization type that suits your ESP project.

> **Note**: 
> - You can reuse existing queries across multiple panels, by selecting **--Dashboard--** as a data source and targeting the panel that contains the existing query.
> - The dashboard that you create references the name of the ESP project. If you rename the ESP project or rename any windows in the ESP project, the dashboard no longer works. As a result, if you want to use the same dashboard with more than one ESP project, you must create a separate dashboard for each project.

### Examples

Some SAS Event Stream Processing Studio examples include Grafana dashboards.

> **Note**:
> - The dashboards have been created using standalone SAS Event Stream Processing that is running in the same namespace as Grafana.
> - If you are using a different environment, such as the SAS Viya platform, you must recreate the queries because the connection URLs will differ.

1. In SAS Event Stream Processing Studio, click ![Help](src/img/icon-helpmenu.png "Help") on any page and select **Examples**.

2. Install the Sailing example project or the Computer Vision with ONNX example project.

3. Run the example project (for example using test mode).

4. Download the Grafana dashboard to your computer:

   - [Dashboard for the Sailing example](https://github.com/sassoftware/esp-studio-examples/tree/main/Advanced/sailing#visualizing-objects-in-grafana)

   - [Dashboard for the Computer Vision with ONNX example](https://github.com/sassoftware/esp-studio-examples/tree/main/Advanced/onnx_object_detection#visualizing-objects-in-grafana)

5. Import the dashboard to Grafana.
   
## Contributing

> We are not currently accepting contributions. 

## License

> This project is licensed under the [Apache 2.0 License](LICENSE).

## Additional Resources
- [Grafana documentation](https://grafana.com/docs/)
- [Grafana tutorials](https://grafana.com/tutorials/)

## Deploying the Plug-in
The plug-in is automatically deployed with the SAS Event Stream Processing app in the Microsoft Marketplace. When SAS Event Stream Processing is deployed with the SAS Viya platform, you can deploy the plug-in by completing the following instructions.
### Prerequisites

* SAS Event Stream Processing running in the SAS Viya platform.
* A Grafana deployment with the name `grafana`. The Grafana deployment can be in the same namespace as SAS Event Stream Processing or in a separate namespace. The deployment scripts provided can install and configure Grafana correctly for you (see below). For more information about which version of Grafana you can use, see [Releases](https://github.com/sassoftware/grafana-esp-plugin/releases).
* Ingress for the Grafana deployment, with the root URL `/grafana`. The deployment scripts provided can install and configure Grafana correctly for you (see below).
* A Linux environment with kubectl installed, to run the plug-in deployment script. 
* Internet access, to enable the plug-in deployment script to download the plug-in from [https://github.com/sassoftware/grafana-esp-plugin/releases](https://github.com/sassoftware/grafana-esp-plugin/releases).
>**Note**: If you have installed SAS VIYA monitoring then the `grafana` deployment and namespace may be elsewhere and manual intervention will be required, it is recommended to run the following scripts as a dry run, and manually copy of the oauth sections of the grafana.ini configuration.


### Deploy a Released Version of the Plug-in

Deployment scripts are provided to deploy the plug-in and configure Grafana. These scripts perform the following tasks:
 * Modify the Grafana deployment by adding the GF_PLUGINS_PREINSTALL_SYNC environment variable to enable Grafana to deploy the plug-in.
 * Create a new `grafana.ini` file to enable OAuth authentication. Creating this file overwrites any existing Grafana configuration.
 * Configure Grafana as an OAuth client with SAS Logon. Users of Grafana are directed to use SAS Logon.
 * Determine the correct Domain Name for your environment by looking at existing ingresses.
 * Optionally automate the installation of Grafana for you.

1. Set the correct Kubernetes configuration file for your environment.
   ```
   export KUBECONFIG=/path/to/kubeconfig
   ```
2. (Optional) Set an environment variable to enable the script to install Grafana for you.
   ```
   export INSTALL_GRAFANA=true

3. (Optional) Set an environment variable to run the script as a dry run to see the resulting configuration and apply the settings manually.
   ```
   export DRY_RUN=true
   ```

4. Run `configure-grafana.sh`, adjusting the command to specify the following variables:
   - The Kubernetes namespace in which SAS Event Stream Processing is deployed, _esp-namespace_.
   - The Kubernetes namespace in which Grafana is deployed, _grafana-namespace_.
   - The _version_ of the plug-in that you want to deploy. Ensure that you specify a version of the plug-in that is compatible with your version of Grafana.
   > **Caution**: Running the script overwrites any existing Grafana configuration.

   ```
   cd ./install
   bash configure-grafana.sh <esp-namespace> <grafana-namespace> <version>
   ```
5. Run `register-oauth-client-viya.sh`, adjusting the command to specify the following variables.
   - The Kubernetes namespace in which SAS Event Stream Processing is deployed, _esp-namespace_.
   - The Kubernetes namespace in which Grafana is deployed, _grafana-namespace_.
   ```
     bash register-oauth-client-viya.sh <esp-namespace> <grafana-namespace>
   ```
6. If Grafana is not running in the same namespace as the SAS Viya platform, you must update the Content Security Policy (CSP) for SAS Logon to allow the Grafana host name to be used as a target of form submission. 
   If you do not update the CSP, the browser blocks the redirect. You can update the CSP in one of the following two ways:
   - Use SAS Environment Manager to update the _content-security-policy_ value under the _sas.commons.web.security_ section.
   - Update the _sas-logon-app_ deployment to add the _SAS_COMMONS_WEB_SECURITY_CONTENTSECURITYPOLICY_ environment variable.

   Update either SAS Environment Manager or the _sas-logon-app_ deployment with the following value, substituting the Grafana host name:
   ```
   default-src 'self'; style-src 'self'; font-src 'self' data:;
   frame-ancestors 'self'; form-action 'self' <grafana-host>;
   ```


### (Optional) Build and Deploy a Privately Signed Version of the Plug-in

Prerequisites:
* You must complete the steps in [Deploy a Released Version of the Plug-in](#deploy-a-released-version-of-the-plug-in).
* Go version 1.23 or above.
* Node version 16 or above.
* Yarn version 1.22 or above

To build and deploy a privately signed version of the plug-in, do the following steps:

1. Build back-end plug-in binaries for Linux, Windows, and Darwin.
   ```
   go run mage.go
   ```
2. Install front-end dependencies.
   ```
   yarn install
   ```
3. Build the plug-in.
   ```
   yarn build
   ```
4. Follow the steps to [privately sign the plug-in](https://grafana.com/docs/grafana/latest/developers/plugins/publish-a-plugin/sign-a-plugin/#sign-a-plugin).
5. Remove the existing plug-in code.
   ```
   kubectl -n <namespace> exec -it <podname> -- /bin/bash -c "rm -r /var/lib/grafana/plugins/sasesp-plugin/*"
   ```
6. Create a plug-in directory from the `dist` directory.
   ```
   cp -r dist sasesp-plugin
   ```
7. Copy the new plug-in code into the Grafana plug-in directory (/var/lib/grafana/plugins) on the pod.
   ```
   kubectl cp ./sasesp-plugin <namespace>/<podname>:/var/lib/grafana/plugins
   ```
   >**Note**: To copy the plug-in code, the Grafana plug-in directory on the Grafana pod must be in persistent storage. Otherwise, the plug-in is lost when the Grafana pod is restarted.
8. Give the Grafana plug-in the correct Execute permissions.
   ````
   kubectl -n <namespace> exec -it <podname> -- /bin/bash -c "chmod 755 /var/lib/grafana/plugins/sasesp-plugin/*"
   ````
9. Stop the Grafana pod.
   ```
   kubectl -n <namespace> scale deployment grafana --replicas=0
   ```
10. Remove the `GF_INSTALL_PLUGINS` environment variable from the Grafana deployment.
    ```
    kubectl -n <namespace> set env deployment/grafana GF_INSTALL_PLUGINS-
    ```
11. Restart the Grafana pod for the changes to take effect.
    ```
    kubectl -n <namespace> scale deployment grafana --replicas=0
    ```
