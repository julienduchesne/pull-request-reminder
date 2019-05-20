FROM alpine

# ca-certificates is needed to download files from S3
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*

# Install the program
ADD pull-request-reminder /app/pull-request-reminder
WORKDIR /app

CMD ["/app/pull-request-reminder"]