FROM wcsiu/tdlib:1.5.0 AS base
FROM golang:1.13 AS golang

COPY --from=base /usr/local/include/td /usr/local/include/td
COPY --from=base /usr/local/lib/libtd* /usr/local/lib/
COPY --from=base /usr/lib/x86_64-linux-gnu/libssl.so.1.1 /usr/lib/x86_64-linux-gnu/libssl.so
COPY --from=base /usr/lib/x86_64-linux-gnu/libcrypto.so.1.1 /usr/lib/x86_64-linux-gnu/libcrypto.so
COPY --from=base /lib/x86_64-linux-gnu/libz.so.1 /lib/x86_64-linux-gnu/libz.so

WORKDIR /demuxer

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go build -o /tmp/demuxer ./cmd/run
# RUN ldd /tmp/demuxer

# FROM gcr.io/distroless/base:latest
# COPY --from=golang /tmp/promoter /
EXPOSE 80
EXPOSE 3000
ENV LD_LIBRARY_PATH="/usr/local/lib/"
ENTRYPOINT [ "/tmp/demuxer" ]
