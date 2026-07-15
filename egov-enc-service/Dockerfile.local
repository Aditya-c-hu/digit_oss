FROM eclipse-temurin:8-jre-alpine
WORKDIR /app
COPY target/egov-enc-service-1.1.4.jar app.jar
EXPOSE 8089
ENTRYPOINT ["java", "-jar", "app.jar"]
