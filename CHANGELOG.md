# Changelog

## 7.43.1

## 7.44.0
### Code Refactoring

* extract handler functions from ESP Websocket client constructor ([f659b7b](https://github.com/sassoftware/grafana-esp-plugin/commit/f659b7bcb7ccd0233e6a04df0f1d8caa2c2b793a))
* improve logging ([3debb3d](https://github.com/sassoftware/grafana-esp-plugin/commit/3debb3d3eeaa0bfdff4cc64fd669dfc167b032b0))
* improve logging ([8ddcd5b](https://github.com/sassoftware/grafana-esp-plugin/commit/8ddcd5bf89a60b4b190ade6e2746f74d2c7ec220))
* reference hashed queries in channel path ([483b02c](https://github.com/sassoftware/grafana-esp-plugin/commit/483b02cf2cd3ebacf22c8bace5c0bcf46377d996))
* replace channelquerystore with more generic synchronous map implementation ([0444544](https://github.com/sassoftware/grafana-esp-plugin/commit/0444544c0051289dd2156e3051023d48a15599d4))
* treat syncmap.Set calls with nil value as Delete calls ([d704aaa](https://github.com/sassoftware/grafana-esp-plugin/commit/d704aaacd8b77f8d45a355c0c508dbe18d474370))

### Bug Fixes

* address an authorization issue when initializing a new datasource ([3a75d63](https://github.com/sassoftware/grafana-esp-plugin/commit/3a75d632caec1427ca9fbd1a3010ba3d7a84aae2))
* improve handling of extra slashes during ESP websocket URL ([04fef0c](https://github.com/sassoftware/grafana-esp-plugin/commit/04fef0c51b945fb79686efcfa8ffa4955c20d561))
* propagate internal ESP URLs for backward compatibility ([73df97b](https://github.com/sassoftware/grafana-esp-plugin/commit/73df97bc9e63c6fe4baaeec243f5b9214ebaf42b))
* revert OAuth passthrough for non-Viya deployments and treat ([3434f08](https://github.com/sassoftware/grafana-esp-plugin/commit/3434f089f7a5cd0baf0d3268d4a21ffb47450201))
* stop premature query calls during fields initialization ([13840bb](https://github.com/sassoftware/grafana-esp-plugin/commit/13840bb32ebcb2e7a10522760270f8dd655b2f0f))
* workaround for compression handling bug in the Grafana SDK's HTTP client ([046eda8](https://github.com/sassoftware/grafana-esp-plugin/commit/046eda80d68e63b3e93a7876f9cf5b779c2d3687))


### Code Refactoring

* extract handler functions from ESP Websocket client constructor ([f659b7b](https://github.com/sassoftware/grafana-esp-plugin/commit/f659b7bcb7ccd0233e6a04df0f1d8caa2c2b793a))
* improve logging ([3debb3d](https://github.com/sassoftware/grafana-esp-plugin/commit/3debb3d3eeaa0bfdff4cc64fd669dfc167b032b0))
* improve logging ([8ddcd5b](https://github.com/sassoftware/grafana-esp-plugin/commit/8ddcd5bf89a60b4b190ade6e2746f74d2c7ec220))
* reference hashed queries in channel path ([483b02c](https://github.com/sassoftware/grafana-esp-plugin/commit/483b02cf2cd3ebacf22c8bace5c0bcf46377d996))
* replace channelquerystore with more generic synchronous map implementation ([0444544](https://github.com/sassoftware/grafana-esp-plugin/commit/0444544c0051289dd2156e3051023d48a15599d4))
* treat syncmap.Set calls with nil value as Delete calls ([d704aaa](https://github.com/sassoftware/grafana-esp-plugin/commit/d704aaacd8b77f8d45a355c0c508dbe18d474370))


## 7.54.0

Update grafana runtime to 10.4.2.

## 7.55.0

Update grafana runtime to 11
Update react to 18
Update grafana compatability to 11.3.0
Fix issue with esp date type fields

## 7.67.1

Update grafana runtime to 12.1.0
Update grafana compatability to 12.2.0
Update install scripts to work with grafana 12.x

## 7.67.0

## 7.68.0

## 7.69.0
