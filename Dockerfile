FROM ubuntu:latest
# liz rice knows best
# https://medium.com/@lizrice/non-privileged-containers-based-on-the-scratch-image-a80105d6d341
RUN useradd -u 10001 scratchuser
RUN apt-get update \
    && apt-get install -y libcap2-bin
COPY nacp /nacp
RUN setcap cap_net_bind_service=+ep /nacp
FROM scratch
COPY --from=0 /nacp /nacp
COPY --from=0 /etc/passwd /etc/passwd
USER scratchuser
ENTRYPOINT ["/nacp"]
