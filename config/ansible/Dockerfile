#
# Ansible
#
FROM ubuntu:latest as ansible
ENV TZ="America/Los_Angeles" DEBIAN_FRONTEND=noninteractive
RUN apt update && \
    apt upgrade -y
RUN apt install -yq tzdata
RUN ln -fs /usr/share/zoneinfo/${TZ} /etc/localtime && \
    echo ${TZ} > /etc/timezone && \
    dpkg-reconfigure --frontend noninteractive tzdata
RUN apt install -yq ansible ansible-lint yamllint git vim dnsutils
RUN groupadd ansible
RUN useradd --shell /usr/bin/bash --home-dir /home/ansible -m -g ansible -G ansible,root ansible
USER ansible
