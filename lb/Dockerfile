FROM golang:1.13 as build
RUN mkdir /app
ADD . /app
WORKDIR /app
ENV GOPROXY=direct
RUN go build -o pr-reviewer-load-balancer ./
FROM gcr.io/distroless/base-debian10
COPY --from=build /app/pr-reviewer-load-balancer /

CMD ["/pr-reviewer-load-balancer"]

EXPOSE 80

