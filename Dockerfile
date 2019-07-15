FROM ubuntu:18.04

# set temporary variables
ARG user=jenkins
ARG group=jenkins
ARG home=/var/lib/jenkins
ARG uid=1000
ARG gid=1000
ARG http_port=8080

# install standard packages
RUN export DEBIAN_FRONTEND=noninteractive && \
    apt update --fix-missing && apt upgrade -y && \
    apt install -y curl sudo locales tzdata openjdk-8-jre-headless \
    apt-transport-https ca-certificates software-properties-common

# generate locale
RUN locale-gen en_US.UTF-8 && locale-gen --no-purge --lang en_US.UTF-8

# set time zone
RUN ln -sf /usr/share/zoneinfo/Europe/Brussels /etc/localtime

# set TERM variable to fix ncurses flickering
ENV TERM=xterm-color

# get latest docker
RUN curl -fsSL https://download.docker.com/linux/ubuntu/gpg | apt-key add -
RUN add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable"
RUN apt update && apt install -y docker-ce

# cleanup apt
RUN rm -rf /var/lib/apt/lists/*

# create a jenkins user and dirs
RUN groupadd -g ${gid} ${group} && \
    useradd -d ${home} -u ${uid} -g ${gid} -m -s /bin/bash ${user} && \
    adduser ${user} sudo && \
    adduser ${user} docker && \
    (echo ${user}:${user} | chpasswd) && \
    chown -R ${user}:${group} ${home}

# set jenkins slave environment variables
ENV SLAVE_IP "localhost"
ENV SLAVE_PORT ${http_port}
ENV SLAVE_WORKDIR ${home}
ENV SLAVE_EXECUTORS "2"

# jenkins runtime parameters
ENV JAVA_OPTS "-Duser.timezone=Europe/Brussels"

VOLUME ${home}
WORKDIR ${home}

#CMD /bin/bash

COPY src/jdsn /usr/local/bin/

# XXX custom program to launch slave
ENTRYPOINT ["/usr/local/bin/jdsn"]
