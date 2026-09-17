// Official @fluent-org/logger (Node.js) client, driven by the functional
// tests.
//
// Unlike the Python and Ruby loggers, this one supports every event mode:
//   FLUENT_EVENT_MODE = Message | Forward | PackedForward | CompressedPackedForward
// and both timestamp formats:
//   FLUENT_EVENT_TIME=1 -> EventTime ext (contemporary)
//   default             -> integer epoch seconds (historical)
// It also supports acknowledgements (FLUENT_ACK=1) and shared key
// authentication (FLUENT_SHARED_KEY).

const { FluentClient, EventTime } = require("@fluent-org/logger");

const host = process.env.FLUENT_HOST || "host.docker.internal";
const port = parseInt(process.env.FLUENT_PORT || "24224", 10);
const tag = process.env.FLUENT_TAG || "testjavascript";
const eventMode = process.env.FLUENT_EVENT_MODE || "PackedForward";
const useEventTime = process.env.FLUENT_EVENT_TIME === "1";

const options = {
  socket: { host, port, timeout: 5000 },
  eventMode,
};
if (process.env.FLUENT_ACK === "1") {
  options.ack = {};
}
if (process.env.FLUENT_SHARED_KEY) {
  options.security = {
    clientHostname: "client.localdomain",
    sharedKey: process.env.FLUENT_SHARED_KEY,
  };
}

const logger = new FluentClient(tag, options);
const record = { client: "javascript", value: 42 };

(async () => {
  if (useEventTime) {
    await logger.emit("event", record, new EventTime(1700000000, 123456789));
  } else {
    await logger.emit("event", record);
  }
  // Give the socket a moment to flush before the process exits.
  await new Promise((resolve) => setTimeout(resolve, 300));
  process.exit(0);
})().catch((err) => {
  console.error(err);
  process.exit(1);
});
