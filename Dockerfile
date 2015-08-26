#
#   Author: Rohith
#   Date: 2015-08-05 01:06:06 +0100 (Wed, 05 Aug 2015)
#
#  vim:ts=2:sw=2:et
#
FROM busybox:latest
MAINTAINER Rohith <gambol99@gmail.com>

ADD https://drone.io/github.com/gambol99/node-register/files/bin/node-register.gz /bin/node-register.gz
RUN gunzip /bin/node-register.gz && \
    chmod +x /bin/node-register

ENTRYPOINT [ "/bin/node-register" ]
