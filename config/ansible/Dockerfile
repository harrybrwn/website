#
# Ansible
#
FROM ubuntu:latest as ansible
ENV TZ="America/Los_Angeles" DEBIAN_FRONTEND=noninteractive
RUN yes | unminimize
RUN apt-get update && \
    apt-get upgrade -y
RUN apt-get install -yq tzdata
RUN ln -fs /usr/share/zoneinfo/${TZ} /etc/localtime && \
    echo ${TZ} > /etc/timezone && \
    dpkg-reconfigure --frontend noninteractive tzdata
RUN apt-get install -yq \
        # ansible          \
        # ansible-lint     \
        python3-pip \
        python3-paramiko \
        python3-boto3    \
        python3-docker   \
        python3-librouteros \
        yamllint \
        git      \
        vim      \
        dnsutils \
        netcat   \
        curl
RUN pip install \
        ansible==9.0.1 \
        ansible-core==2.16.0

RUN mkdir -p /usr/local/bin && \
    curl -sSL -o /usr/local/bin/kubectl "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" && \
    chmod +x /usr/local/bin/kubectl && \
    groupadd ansible && \
    useradd \
        --shell /usr/bin/bash \
        --home-dir /home/ansible \
        -m \
        -g ansible \
        -G ansible,root \
        ansible
RUN --mount=type=ssh \
    mkdir /home/ansible/.ssh && \
    ssh-keyscan 10.0.0.1 >> /home/ansible/.ssh/known_hosts && \
    echo '10.0.0.1 ssh-rsa AAAAB3NzaC1yc2EAAAABAwAAAQEAmiNE3hZ1Iiis4WYTnc183jvYTKqfYv5KRxZsWzxayugOn2r7IZpuJ23X5biRcXc8ldSDdO4Gv1xplf9FQVERJCRwRX2CnhBC/k4Vf+M2JQDCKCgXyfMWjtYT5PSgpnH+SwE7Uw1HkMceJiuxGHl+pdoHlZOyt9eIBh3rjF1GpqA+6e8mNYhpcom7XXQRvaMilyjvZGbhuBkInJCy1PgXqNkst83WUxSZhSrGbvlvEIPqZsn6pyxVXKiROv6uO+Bsro3XoJs2WO3QaYDUKy2lbHK2YcufegIPtIOOZ6mQXaVXEdwJ9Ah0jfAuWslSjqrAWLrYbX/JOvWhvmylfyVFew==' >> /home/ansible/.ssh/known_hosts && \
    echo 'router.lan ssh-rsa AAAAB3NzaC1yc2EAAAABAwAAAQEAmiNE3hZ1Iiis4WYTnc183jvYTKqfYv5KRxZsWzxayugOn2r7IZpuJ23X5biRcXc8ldSDdO4Gv1xplf9FQVERJCRwRX2CnhBC/k4Vf+M2JQDCKCgXyfMWjtYT5PSgpnH+SwE7Uw1HkMceJiuxGHl+pdoHlZOyt9eIBh3rjF1GpqA+6e8mNYhpcom7XXQRvaMilyjvZGbhuBkInJCy1PgXqNkst83WUxSZhSrGbvlvEIPqZsn6pyxVXKiROv6uO+Bsro3XoJs2WO3QaYDUKy2lbHK2YcufegIPtIOOZ6mQXaVXEdwJ9Ah0jfAuWslSjqrAWLrYbX/JOvWhvmylfyVFew==' >> /home/ansible/.ssh/known_hosts
RUN \
	# Mikrotik RouterOS
    # https://docs.ansible.com/ansible/devel/collections/community/routeros/
    ansible-galaxy collection install \
        community.general:8.0.2 \
        community.routeros:2.11.0
USER ansible

