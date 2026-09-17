# Official Java client (org.fluentd:fluent-logger) for the functional tests.
FROM maven:3.9-eclipse-temurin-21

WORKDIR /client
COPY java.pom.xml /client/pom.xml
COPY java.client.java /client/src/main/java/client/Main.java

# Compile and record the runtime classpath, so the entrypoint stays fast.
RUN mvn -q -B compile dependency:build-classpath -Dmdep.outputFile=/client/classpath.txt

ENTRYPOINT ["sh", "-c", "exec java -cp /client/target/classes:$(cat /client/classpath.txt) client.Main"]
