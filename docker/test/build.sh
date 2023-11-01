mkdir -p data/grafana-oss/plugins/
rm -rf data/grafana-oss/plugins/sasesp-plugin/dist
ln -srf ../../dist data/grafana-oss/plugins/sasesp-plugin

mkdir -p data/grafana-oss/public/maps/

docker build .