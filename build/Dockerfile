FROM alpine:3.8

ENV OPERATOR=/usr/local/bin/SnapScheduler \
    USER_UID=1001 \
    USER_NAME=SnapScheduler

# install operator binary
COPY build/_output/bin/SnapScheduler ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}

ARG builddate="(unknown)"
ARG version="(unknown)"

LABEL build-date="${builddate}"
LABEL io.k8s.description="Operator to manage scheduled PV snapshots"
LABEL name="SnapScheduler"
LABEL Summary="Operator to manage scheduled PV snapshots"
LABEL vcs-type="git"
LABEL vcs-url="https://github.com/backube/SnapScheduler"
LABEL vendor="Backube"
LABEL version="${version}"
