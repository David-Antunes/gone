FROM public.ecr.aws/docker/library/golang:1.23 AS build

WORKDIR /gone

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY main.go .
COPY internal internal
COPY api api
RUN GOAMD64=v3 go build 


FROM alpinelinux/docker-cli

RUN apk add --no-cache iputils-ping ethtool gcompat

COPY --from=build /gone/gone /gone


CMD ["/gone"]
