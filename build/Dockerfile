FROM registry.access.redhat.com/ubi7/ubi-minimal:latest

ENV OPERATOR=/usr/local/bin/SnapScheduler \
    USER_UID=1001 \
    USER_NAME=SnapScheduler

# install operator binary
COPY build/_output/bin/SnapScheduler ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  chmod a+r /usr/local/bin/entrypoint && \
     /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}

ARG builddate="(unknown)"
ARG description="Operator to manage scheduled PV snapshots"
ARG version="(unknown)"

LABEL build-date="${builddate}"
LABEL description="${description}"
LABEL io.k8s.description="${description}"
LABEL io.k8s.displayname="SnapScheduler: A snapshot scheduler"
LABEL name="SnapScheduler"
LABEL summary="${description}"
LABEL vcs-type="git"
LABEL vcs-url="https://github.com/backube/SnapScheduler"
LABEL vendor="Backube"
LABEL version="${version}"
