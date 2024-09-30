FROM ubuntu AS BUILD

WORKDIR /gone
RUN apt update && apt install -y software-properties-common \
&& add-apt-repository ppa:longsleep/golang-backports \
&& apt install -y golang-go ca-certificates curl \
&& install -m 0755 -d /etc/apt/keyrings \
&& curl -fsSL https://download.docker.com/linux/debian/gpg -o /etc/apt/keyrings/docker.asc \
&& chmod a+r /etc/apt/keyrings/docker.asc

# Add the repository to Apt sources:
RUN echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu \
  $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
  tee /etc/apt/sources.list.d/docker.list > /dev/null
RUN apt update && apt install -y docker-ce-cli net-tools iputils-ping ethtool

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY main.go .
COPY internal internal
COPY api api
RUN go build 

CMD ["./gone"]
