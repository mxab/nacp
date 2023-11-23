FROM grafana/agent:v0.37.4

ENV AGENT_MODE=flow
COPY grafana-agent-flow.river /etc/grafana-agent-flow.river

EXPOSE 12345
CMD [ "run", "--server.http.listen-addr=0.0.0.0:12345", "/etc/grafana-agent-flow.river" ]
