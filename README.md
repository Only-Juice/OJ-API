# OJ-API

[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/Only-Juice/OJ-API)

## Introduction
OJ-API is a powerful and flexible API designed to handle various operations. This README will guide you through the process of building and running the OJ-API using Docker.

## Prerequisites
- Docker installed on your machine.

## Building the Docker Image
To build the Docker image for OJ-API, navigate to the project directory and run the following command:
```sh
docker build -t ojapi . --no-cache
```

## Running the Docker Container
Once the image is built, you can run the Docker container with the following command:
```sh
docker run --privileged -p 3001:3001 ojapi
```

This will start the OJ-API and map port 3001 of the container to port 3001 on your host machine.

## Accessing the API
After running the container, the API will be accessible at `http://localhost:3001`.

## Conclusion
You have successfully built and run the OJ-API using Docker. For more information, please refer to the project's documentation.
