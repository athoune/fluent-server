const FluentServer = require("@fluent-org/logger").FluentServer;

const server = new FluentServer(
    {
        listenOptions: { 
            host: "127.0.0.1",
            port: 24224 },
        security: {
            serverHostname: "server.example.com",
            sharedKey: "plop"
        }
    }
);

server.on("entry", (tag, time, record) => {
    console.log(tag, time, record);
});

    server.listen();



