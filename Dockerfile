FROM scratch

ADD pull-request-reminder /app/pull-request-reminder
WORKDIR /app

CMD ["/app/pull-request-reminder"]