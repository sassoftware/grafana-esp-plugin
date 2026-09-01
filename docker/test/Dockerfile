FROM grafana/grafana:13.2.0

USER root
WORKDIR /root

ADD data/grafana-oss/plugins/sasesp-plugin /var/lib/grafana/plugins/sasesp-plugin
COPY config/config.ini /etc/grafana/grafana.ini

#docker images -a
#docker run -p 3000:3000 <image-id>
#http://localhost:3000/grafana
#login admin/admin
