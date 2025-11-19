FROM golang:alpine AS build

RUN mkdir /app
COPY . /app
WORKDIR /app
RUN go tool task build

FROM alpine AS run

COPY --from=build /app/bin/namelist-fetch /namelist-fetch

CMD [ "/namelist-fetch" ]