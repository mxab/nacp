FROM scratch

COPY nacp /
ENTRYPOINT ["/nacp"]
