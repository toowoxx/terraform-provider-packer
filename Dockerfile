FROM ubuntu:20.04

RUN apt update -y && apt upgrade -y
RUN apt install -y software-properties-common apt-utils
RUN add-apt-repository -y ppa:longsleep/golang-backports && apt update -y
RUN apt install -y golang bash make ca-certificates
