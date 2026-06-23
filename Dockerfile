FROM golang:1.26 as go-builder
WORKDIR /src
COPY Makefile /src
RUN make go
COPY . /src
RUN make go-static

FROM nginx:1

RUN sed -i 's@}@application/wasm wasm;}@' /etc/nginx/mime.types

COPY --from=go-builder /src/server/public/ /usr/share/nginx/html
RUN test -f /usr/share/nginx/html/index.html
