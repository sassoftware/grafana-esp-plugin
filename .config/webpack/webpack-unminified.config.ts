/*
	Copyright © 2023, SAS Institute Inc., Cary, NC, USA.  All Rights Reserved.
	SPDX-License-Identifier: Apache-2.0
*/

// Extends the base Grafana webpack config to disable minification.
import type { Configuration } from 'webpack';
import { merge } from 'webpack-merge';
import grafanaConfig from './webpack.config';

const config = async (env): Promise<Configuration> => {
  const baseConfig = await grafanaConfig(env);

  return merge(baseConfig, {
    optimization: {
      minimize: false,
    },
  });
};

export default config;
