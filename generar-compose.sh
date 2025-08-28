#!/bin/bash

if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <output_file> <number_of_clients>"
    exit 1
fi

OUTPUT_FILE=$1
NUM_CLIENTS=$2

if ! [[ "$NUM_CLIENTS" =~ ^[0-9]+$ ]]; then
    echo "Error: Number of clients must be a non-negative integer"
    exit 1
fi

cat > "$OUTPUT_FILE" <<EOL
name: tp0
services:
  server:
    container_name: server
    image: server:latest
    entrypoint: python3 /main.py
    environment:
      - PYTHONUNBUFFERED=1
      - LOGGING_LEVEL
    volumes:
      - ./server/config.ini:/config.ini
    networks:
      - testing_net

EOL

for ((i=1; i<=NUM_CLIENTS; i++)); do
    cat >> "$OUTPUT_FILE" <<EOL
  client$i:
    container_name: client$i
    image: client:latest
    entrypoint: /client
    environment:
      - CLI_ID=$i
      - CLI_LOG_LEVEL
    volumes:
      - ./client/config.yaml:/config.yaml
    networks:
      - testing_net
    depends_on:
      - server

EOL
done

cat >> "$OUTPUT_FILE" <<EOL
networks:
  testing_net:
    ipam:
      driver: default
      config:
        - subnet: 172.25.125.0/24
EOL

echo "Docker Compose file '$OUTPUT_FILE' generated successfully with $NUM_CLIENTS clients."

chmod +x "$OUTPUT_FILE"
