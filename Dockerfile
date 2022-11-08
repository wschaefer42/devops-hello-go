FROM golang:1.19-alpine AS build

WORKDIR /src
COPY main.go go.* ./
COPY views ./views
RUN CGO_ENABLED=0 go build -o hello-app
RUN ls -R

FROM scratch
WORKDIR /app/
COPY --from=build /src/hello-app hello-app
COPY --from=build /src/views ./views
ENV PORT=8001
ENV HOST=0.0.0.0
ENTRYPOINT ["/app/hello-app"]