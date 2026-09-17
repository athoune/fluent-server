# Official Ruby client (fluent-logger) for the functional tests.
FROM ruby:3.4-slim

# fluent-logger pulls the msgpack gem, which builds a native extension.
RUN apt-get update \
    && apt-get install -y --no-install-recommends build-essential \
    && rm -rf /var/lib/apt/lists/* \
    && gem install fluent-logger -v 0.11.0 --no-document

COPY ruby.client.rb /client.rb

ENTRYPOINT ["ruby", "/client.rb"]
