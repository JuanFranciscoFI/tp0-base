#!/bin/bash

if ! docker ps --format '{{.Names}}' | grep -q '^server$'; then
    echo "action: test_echo_server | result: fail"
    exit 1
fi

SERVER_PORT=$(docker exec server cat /config.ini 2>/dev/null | grep -i '^SERVER_PORT' | grep -oE '[0-9]+' || echo "12345")

NETWORK=$(docker inspect -f '{{range $net,$v := .NetworkSettings.Networks}}{{$net}}{{end}}' server 2>/dev/null)

if [ -z "$NETWORK" ]; then
    echo "action: test_echo_server | result: fail"
    exit 1
fi

TEST_MSG="test"

if RESPONSE=$(docker run --rm --network "$NETWORK" busybox sh -c "echo -n '$TEST_MSG' | nc -w 2 server $SERVER_PORT" 2>/dev/null); then
    RESPONSE=$(echo "$RESPONSE" | tr -d '\r' | tr -d '\n')
    
    if [ "$RESPONSE" = "$TEST_MSG" ]; then
        echo "action: test_echo_server | result: success"
    elif [[ "$RESPONSE" == error* ]]; then
        echo "action: test_echo_server | result: fail"
    else
        echo "action: test_echo_server | result: fail"
    fi
else
    echo "action: test_echo_server | result: fail"
fi

exit 0
