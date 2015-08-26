#
#   Author: Rohith
#   Date: 2015-08-05 01:06:06 +0100 (Wed, 05 Aug 2015)
#
#  vim:ts=2:sw=2:et
#
FROM busybox:latest
MAINTAINER Rohith <gambol99@gmail.com>

ADD https://github.com/gambol99/node-register/releases/download/v0.0.3/node-register_0.0.3_linux_x86_64.gz /bin/node-register.gz
RUN gunzip /bin/node-register.gz && \
    chmod +x /bin/node-register

ENTRYPOINT [ "/bin/node-register" ]
