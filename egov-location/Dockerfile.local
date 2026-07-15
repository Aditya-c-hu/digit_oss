FROM eclipse-temurin:8-jre-alpine
WORKDIR /app
COPY target/egov-location-1.1.5-SNAPSHOT.jar app.jar
EXPOSE 8082
ENTRYPOINT ["java", "-jar", "app.jar"]
