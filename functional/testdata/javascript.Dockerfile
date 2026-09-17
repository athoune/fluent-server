# Official Node.js client (@fluent-org/logger) for the functional tests.
FROM node:22-slim

WORKDIR /client
RUN npm install @fluent-org/logger@1.0.10

COPY javascript.client.js /client/client.js

ENTRYPOINT ["node", "/client/client.js"]
