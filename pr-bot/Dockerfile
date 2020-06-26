FROM golang:1.13-buster as build
RUN mkdir /app
ADD . /app
WORKDIR /app
ENV GOPROXY=direct
RUN go build -o project-bot-api ./
FROM gcr.io/distroless/base-debian10
COPY --from=build /app/project-bot-api /

CMD ["/project-bot-api"]

EXPOSE 80
