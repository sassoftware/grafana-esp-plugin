# Instructions on how to set up this plugin for development using docker containers.

## Run the grafana server

`npm run server`

To build your plugin with debug information and start the debugger
...

Connect to bash inside the running container:

`docker exec -ti dev-grafana-plugin-dev bash`

Inside the container run the debug script which will rebuild the plugin and open the debugger (this may look frozen but its mage/go doing its build):

`cd /var/lib/grafana/plugins/sasesp-plugin`
`bash start-debug.sh`

If the script fails run command individually:

`mage build:debug`
`mage reloadPlugin`
`dlv attach --headless --api-version 2 --accept-multiclient --listen=:3222 $(pgrep -f sasesp-plugin)`

In vscode run the `Debug in Container` configuration to attach to the debugger

To recompile client code on change run
`npm run dev`

To test grafana plugin navigate to:

`http://localhost:3000/`
admin/admin


## Create a grafana docker image with the plugin inside

Follow the top level README.md instructions to build the grafana plugin.

`npm run server-test` 

Tag and push the docker container to test on another environment.

`docker images`
Grab the latest image ID

`docker tag dev-grafana-plugin-dev registry.unx.sas.com/esp-client/kindviya:grafana-mtlYOU-1`

`docker push registry.unx.sas.com/esp-client/kindviya:grafana-mtlYOU-1`
