# Change this variable to the name of the app
ARG GO_VERSION=1.20.6

######################################

FROM golang:$GO_VERSION-alpine as builder

WORKDIR /endless_book

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -o endless_book .

######################################

FROM scratch

COPY --from=builder /endless_book/endless_book .

EXPOSE 8080

ENTRYPOINT [ "./endless_book" ]
