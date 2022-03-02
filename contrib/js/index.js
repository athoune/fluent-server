const FluentClient = require("@fluent-org/logger").FluentClient;
const http = require("http");

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
    const magic = Math.random();
    await logger.emit("my_tag", {
      name: "Bob",
      age: 42,
      magic: magic,
    });
    http.get("http://127.0.0.1:24280", (res) => {
        res.setEncoding('utf8');
        let rawData = '';
        res.on('data', (chunk) => { rawData += chunk; });
        res.on('end', () => {
            const parsedData = JSON.parse(rawData);
            //console.log(parsedData);
            let found = false;
            for (evt of parsedData["tag_prefix.my_tag"]) {
                if (evt.record.magic == magic) {
                    found = true;
                }
            }
            if (! found) {
                throw "Event not found in the mirror server";
            }
            console.log("Test passed");
        });
    });
};

demo().then(() => {
    console.log("demo end");
});
