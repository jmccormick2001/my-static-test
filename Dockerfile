FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

ENV CUSTOM_TEST=my-static-test \
    USER_UID=1001 \
    USER_NAME=testuser

COPY bin/user_setup /usr/local/bin/
COPY bin/my-static-test /usr/local/bin/
COPY bin/entrypoint /usr/local/bin/

RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
