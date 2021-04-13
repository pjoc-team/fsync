FROM golang:latest as build

ARG REPOSITORY
ARG GOPROXY
ENV BUILD_PROJECT_PATH=${GOPATH}/src/${REPOSITORY}
ENV GO111MODULE=on
ENV GOPROXY=${GOPROXY}
ENV BIN=/app/bin

ADD . /tmp
RUN export REPOSITORY=`cat /tmp/go.mod | grep -E "^module\s[0-9a-zA-Z\./_\-]+" | awk '{print $2}'`; \
    export NAME=`basename $REPOSITORY`; \
    export APP=`basename $REPOSITORY`; \
    export BUILD_PROJECT_PATH="${GOPATH}/src/${REPOSITORY}"; \
    env; \
    if [ -z "$REPOSITORY" ]; then \
        echo "repository arg is null!"; \
        exit 1; \
    else \
        echo "path===${GOPATH}/src/$REPOSITORY"; \
    fi; \
    mkdir -p "${BUILD_PROJECT_PATH}"; \
    cp -R /tmp/* ${BUILD_PROJECT_PATH}; \
    cd ${BUILD_PROJECT_PATH}; \
    pwd; \
    if [ -f "go_build.sh" ]; then \
        bash go_build.sh; \
        ls /app/bin; \
        mv /app/bin/* /app/; \
        mv /app/${APP} /app/main; \
    else \
        echo "not found go_build.sh. files: `ls`" ;\
    fi

RUN env


FROM alpine:latest as certs
RUN apk --update add ca-certificates && \
    apk add bash && \
    mkdir -p /app

#ENV APP=$app

COPY --from=build /app/ /app/

WORKDIR /app
CMD ["bash", "-c", "/app/main"]
EXPOSE 8080
