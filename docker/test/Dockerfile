FROM grafana/grafana:9.5.13-ubuntu

USER root
WORKDIR /root

ADD data/grafana-oss/plugins/sasesp-plugin /var/lib/grafana/plugins/sasesp-plugin
COPY config/config.ini /etc/grafana/grafana.ini