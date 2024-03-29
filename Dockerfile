FROM debian:bullseye as build

RUN apt-get -y update && \
    apt-get install -y --no-install-recommends git unzip

COPY . /go/src/gitlab.com/olaris/olaris-server
WORKDIR /go/src/gitlab.com/olaris/olaris-server

RUN make download-olaris-react generate build-local

FROM debian:bullseye AS release

# Install sudo because entrypoint.sh uses it
RUN apt-get -y update && \
    apt-get install -y --no-install-recommends sudo ca-certificates && \
    apt-get install -y ffmpeg && \
    apt-get autoremove && apt-get clean

RUN useradd --create-home -U olaris

COPY --from=build /go/src/gitlab.com/olaris/olaris-server/build/olaris /opt/olaris/olaris
COPY ./docker/entrypoint.sh /
RUN mkdir -p /home/olaris/.config/olaris && chown olaris:olaris /home/olaris/.config/olaris
VOLUME /home/olaris/.config/olaris
EXPOSE 8080
ENTRYPOINT ["/entrypoint.sh", "/opt/olaris/olaris"]
