FROM alpine:3.22

RUN apk add --no-cache curl tzdata
COPY crontab /etc/crontabs/root
COPY run-digest.sh /usr/local/bin/run-digest.sh
RUN chmod 0755 /usr/local/bin/run-digest.sh

CMD ["crond", "-f", "-l", "2"]
