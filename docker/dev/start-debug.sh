#!/bin/bash
cd /var/lib/grafana/plugins/sasesp-plugin && mage build:debug && mage reloadPlugin && dlv attach --headless --api-version 2 --accept-multiclient --listen=:3222 $(pgrep -f sasesp-plugin)
