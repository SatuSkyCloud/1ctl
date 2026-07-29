FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod main.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /model-api .

FROM scratch
COPY --from=build /model-api /model-api
COPY model/model-v1.json /model/model.json
USER 10001:10001
EXPOSE 8080
ENTRYPOINT ["/model-api"]
