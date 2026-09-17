#!/usr/bin/env ruby
# frozen_string_literal: true

# Official fluent-logger (Ruby) client, driven by the functional tests.
#
# fluent-logger always uses Message mode and packs [tag, timestamp, record].
# The nanosecond_precision switch selects the timestamp format:
#   - false (default): integer epoch seconds, the historical format
#   - true: EventTime ext, the contemporary format

require 'fluent-logger'

host = ENV.fetch('FLUENT_HOST', 'host.docker.internal')
port = ENV.fetch('FLUENT_PORT', '24224').to_i
tag = ENV.fetch('FLUENT_TAG', 'testruby')
nanosecond_precision = ENV.fetch('FLUENT_NANOSECOND_PRECISION', '0') == '1'

log = Fluent::Logger::FluentLogger.new(nil,
                                       host: host,
                                       port: port,
                                       nanosecond_precision: nanosecond_precision)

unless log.post("#{tag}.event", { 'client' => 'ruby', 'value' => 42 })
  warn "post failed: #{log.last_error.inspect}"
  exit 1
end

log.close if log.respond_to?(:close)
