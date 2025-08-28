#!/bin/bash

if ! docker ps --format '{{.Names}}' | grep -q '^server$'; then
    echo "action: test_echo_server | result: fail"
    exit 1
fi

NETWORK=$(docker inspect -f '{{range $net,$v := .NetworkSettings.Networks}}{{$net}}{{end}}' server 2>/dev/null)

if [ -z "$NETWORK" ]; then
    echo "action: test_echo_server | result: fail"
    exit 1
fi

PORT=12345
TEST_MSG="test"

RESPONSE=$(docker run --rm --network "$NETWORK" alpine sh -c "echo '$TEST_MSG' | nc server $PORT" 2>/dev/null)

if [ "$RESPONSE" = "$TEST_MSG" ]; then
    echo "action: test_echo_server | result: success"
else
    echo "action: test_echo_server | result: fail"
fi
