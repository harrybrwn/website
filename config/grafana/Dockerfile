ARG GRAFANA_VERSION
FROM grafana/grafana:${GRAFANA_VERSION}
USER root
COPY scripts/wait.sh /usr/local/bin/wait.sh
COPY config/grafana/grafana.ini     /etc/grafana/grafana.ini
COPY config/grafana/datasources.yml /etc/grafana/provisioning/datasources/datasource.yml
COPY config/grafana/dashboards.yml  /etc/grafana/provisioning/dashboards/dashboards.yml
COPY config/grafana/dashboards/     /etc/grafana/provisioning/dashboards/dashboards/