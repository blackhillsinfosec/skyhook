FROM amd64/alpine:3.16
RUN apk update && apk add bash iproute2 libc6-compat curl
COPY --chmod=711 skyhook/skyhook.elf /sbin/skyhook
HEALTHCHECK --interval=1m --timeout=3s \
  CMD curl --insecure https://$(ip a show eth0 | grep inet | head -n 1 | awk '{print $2}' | sed -r -e 's/\/.+//') || exit 1
VOLUME ["/skyhook-config"]
EXPOSE 443/tcp 65000/tcp
ENTRYPOINT skyhook server run -c /skyhook-config/config.yml
