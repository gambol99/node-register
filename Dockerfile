#
#   Author: Rohith
#   Date: 2015-08-05 01:06:06 +0100 (Wed, 05 Aug 2015)
#
#  vim:ts=2:sw=2:et
#
FROM busybox:latest
MAINTAINER Rohith <gambol99@gmail.com>

ADD bin/node-register /bin/node-register
RUN chmod +x /bin/node-register

ENTRYPOINT [ "/bin/node-register" ]
