FROM grafana/grafana:11.3.0-ubuntu

USER root
WORKDIR /root

ADD data/grafana-oss/plugins/sasesp-plugin /var/lib/grafana/plugins/sasesp-plugin
COPY config/config.ini /etc/grafana/grafana.ini

#docker images
#docker run -p 3000:3000 <image-id> 
#http://localhost:3000/grafana
#login admin/admin