FROM wcsiu/tdlib:1.5.0 AS base
FROM golang:1.13 AS golang

COPY --from=base /usr/local/include/td /usr/local/include/td
COPY --from=base /usr/local/lib/libtd* /usr/local/lib/
COPY --from=base /usr/lib/x86_64-linux-gnu/libssl.a /usr/lib/x86_64-linux-gnu/libssl.a
COPY --from=base /usr/lib/x86_64-linux-gnu/libcrypto.a /usr/lib/x86_64-linux-gnu/libcrypto.a
COPY --from=base /usr/lib/x86_64-linux-gnu/libz.a /usr/lib/x86_64-linux-gnu/libz.a

WORKDIR /demuxer

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go build --ldflags "-extldflags '-static -L/usr/local/lib -ltdjson_static -ltdjson_private -ltdclient -ltdcore -ltdactor -ltddb -ltdsqlite -ltdnet -ltdutils -ldl -lm -L/usr/lib/x86_64-linux-gnu -lssl -lcrypto -lstdc++ -L/lib/x86_64-linux-gnu -lz'" -o /tmp/demuxer ./cmd/run
# RUN ldd /tmp/demuxer

FROM gcr.io/distroless/base:latest
COPY --from=golang /tmp/demuxer /demuxer-runner
COPY --from=golang /demuxer/configs/configs.yml /configs/configs.yml
EXPOSE 80
EXPOSE 3000
ENTRYPOINT [ "/demuxer-runner" ]
