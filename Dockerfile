FROM golang:1.12.6-stretch

RUN apt update && apt install -y libxml2-dev
COPY ./vendor /one/vendor
COPY ./go.mod ./go.sum /one/
COPY ./internal /one/internal
COPY ./main.go /one
WORKDIR /one
RUN GOOS=linux go build -a -mod vendor -o /bin/one

EXPOSE 8080
ENTRYPOINT ["/bin/one"]
