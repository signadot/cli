ARG USER=signadot
ARG GOARCH=amd64
FROM ubuntu
ARG USER
ARG GOARCH

RUN apt update
RUN apt upgrade -y
RUN apt-get install -y ca-certificates
RUN apt-get install -y iptables
RUN apt-get install -y sudo
RUN apt-get install -y curl
RUN apt-get install -y vim
RUN apt-get install -y git
RUN apt-get install -y python3
RUN apt-get install -y python3.10-venv
RUN apt-get install -y unzip
RUN apt-get install -y wget

COPY install-aws.sh /install-aws.sh
RUN /install-aws.sh
COPY install-go.sh /install-go.sh
RUN GOARCH=$GOARCH /install-go.sh
RUN /usr/local/go/bin/go install github.com/goreleaser/goreleaser@latest

COPY signadot /usr/bin/signadot



RUN adduser --disabled-password $USER
RUN adduser $USER sudo
RUN echo '%sudo ALL=(ALL) NOPASSWD:ALL' >> /etc/sudoers


USER $USER
WORKDIR /home/$USER

ENTRYPOINT ["/bin/bash"]

