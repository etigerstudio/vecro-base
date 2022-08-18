FROM alpine:3.14
COPY vecro-base /
WORKDIR /
ENTRYPOINT ["./vecro-base"]