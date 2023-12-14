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
RUN apt-get install -y protobuf-compiler

COPY signadot /usr/bin/signadot

RUN adduser --disabled-password $USER
RUN adduser $USER sudo
RUN echo '%sudo ALL=(ALL) NOPASSWD:ALL' >> /etc/sudoers


USER $USER
WORKDIR /home/$USER

CMD ["/bin/bash"]
ENTRYPOINT ["/bin/bash"]

