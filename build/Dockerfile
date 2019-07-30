FROM wcsiu/tdlib:1.4.0 AS base
FROM golang:1.12.7 AS golang

COPY --from=base /usr/local/include/td /usr/local/include/td
COPY --from=base /usr/local/lib/libtd* /usr/local/lib/
COPY --from=base /usr/lib/x86_64-linux-gnu/libssl.so.1.1 /usr/lib/x86_64-linux-gnu/libssl.so
COPY --from=base /usr/lib/x86_64-linux-gnu/libcrypto.so.1.1 /usr/lib/x86_64-linux-gnu/libcrypto.so
COPY --from=base /lib/x86_64-linux-gnu/libz.so.1 /lib/x86_64-linux-gnu/libz.so

WORKDIR /promoter

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go build -o /tip ./cmd/run

EXPOSE 80
ENV LD_LIBRARY_PATH="/usr/local/lib/"
ENTRYPOINT [ "/tip" ]
