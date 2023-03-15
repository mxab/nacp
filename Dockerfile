FROM ubuntu:latest
# liz rice knows best
# https://medium.com/@lizrice/non-privileged-containers-based-on-the-scratch-image-a80105d6d341
RUN useradd -u 10001 scratchuser
FROM scratch
COPY nacp /nacp
COPY --from=0 /etc/passwd /etc/passwd
USER scratchuser
ENTRYPOINT ["/nacp"]
