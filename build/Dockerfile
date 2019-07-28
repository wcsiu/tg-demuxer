FROM amd64/ubuntu:18.04 AS base

WORKDIR /build

RUN apt-get update && \
    apt-get upgrade -y && \
    apt-get install -y git cmake build-essential gperf libssl-dev zlib1g-dev

RUN git clone https://github.com/tdlib/td.git && \
    ls && \
    cd td && \
    mkdir build && \
    cd build && \
    cmake -DCMAKE_BUILD_TYPE=Release .. && \
    cmake --build . && \
    make install

FROM golang:1.12.7 AS golang

COPY --from=base /usr/local/include/td /usr/local/include/td
COPY --from=base /usr/local/lib/libtd* /usr/local/lib/

WORKDIR /promoter

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN echo "/usr/local/lib" > /etc/ld.so.conf.d/usr_local_lib.conf && ldconfig

RUN go build -o /tmp/promoter ./cmd/run

FROM gcr.io/distroless/base:latest
COPY --from=golang /tmp/promoter /
EXPOSE 80
ENTRYPOINT [ "/promoter" ]
