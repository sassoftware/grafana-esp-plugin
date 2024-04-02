FROM grafana/grafana:10.4.1-ubuntu

USER root
WORKDIR /root

ADD data/grafana-oss/plugins/sasesp-plugin /var/lib/grafana/plugins/sasesp-plugin
COPY config/config.ini /etc/grafana/grafana.ini