const FluentClient = require("@fluent-org/logger").FluentClient;

const logger = new FluentClient("tag_prefix", {
  socket: {
    host: "127.0.0.1",
    port: 24224,
    timeout: 1000, // 1 second
  },
  eventMode: "Forward",
  //ack: {},
});


async function demo() {
    await logger.emit("my_tag", {name: "Bob", age: 42});
};

demo().then(() => {
    console.log("demo end");
});
