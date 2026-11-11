FROM golang:1.22
MAINTAINER Filip Kopecky
WORKDIR /src

RUN apt-get update
RUN apt-get install -y curl

COPY . .
RUN go build -o /usr/local/bin/api .

EXPOSE 8080
CMD /usr/local/bin/api
