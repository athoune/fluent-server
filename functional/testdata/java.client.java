package client;

import java.util.HashMap;
import java.util.Map;

import org.fluentd.logger.FluentLogger;

/**
 * Official fluent-logger (Java) client, driven by the functional tests.
 *
 * fluent-logger-java uses Message mode and packs [tag, timestamp, record],
 * with an integer epoch timestamp in seconds.
 */
public final class Main {
    private Main() {
    }

    public static void main(String[] args) {
        String host = System.getenv().getOrDefault("FLUENT_HOST", "host.docker.internal");
        int port = Integer.parseInt(System.getenv().getOrDefault("FLUENT_PORT", "24224"));
        String tag = System.getenv().getOrDefault("FLUENT_TAG", "testjava");

        FluentLogger logger = FluentLogger.getLogger(tag, host, port);

        Map<String, Object> record = new HashMap<>();
        record.put("client", "java");
        record.put("value", 42);

        if (!logger.log("event", record)) {
            System.err.println("log returned false");
            System.exit(1);
        }
        logger.flush();
        FluentLogger.closeAll();
    }
}
