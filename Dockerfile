FROM golang:1.16.3 AS build-env

WORKDIR /src
ADD . /src

RUN make

FROM ubuntu
COPY --from=build-env /src/grit /
ENTRYPOINT ["/grit"]
