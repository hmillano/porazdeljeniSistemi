#!/bin/bash

if [ "$1" == "" ]; then
    echo "Usage: $0 <number_of_nodes>"
    exit 1
fi

NUM_NODES=$1
BASE_PORT=9876

if [ $NUM_NODES -lt 1 ]; then
    echo "Number of nodes must be at least 1"
    exit 1
fi

echo "Starting chain replication with $NUM_NODES nodes"

LAST_NODE=$((NUM_NODES - 1))

go build -o chain_replication .

rm -f pids.txt
rm -f server_*.log

wait_for_port() {
    local port=$1
    local retry=0
    local max_retries=30
    
    while ! nc -z $(hostname) $port && [ $retry -lt $max_retries ]; do
        sleep 0.1
        retry=$((retry + 1))
    done
    
    if [ $retry -eq $max_retries ]; then
        echo "Timeout waiting for port $port"
        return 1
    fi
    return 0
}

start_node() {
    local node_id=$1
    local node_type
    
    if [ $node_id -eq 0 ]; then
        node_type="HEAD"
    elif [ $node_id -eq $LAST_NODE ]; then
        node_type="TAIL"
    else
        node_type="MIDDLE"
    fi
    
    local port=$((BASE_PORT + node_id))
    echo "Starting $node_type node (ID: $node_id)"
    echo "Listening on port: $port"
    
    if [ $node_id -gt 0 ]; then
        echo "Previous node URL: $(hostname):$((port-1))"
    fi
    
    if [ $node_id -lt $LAST_NODE ]; then
        echo "Next node URL: $(hostname):$((port+1))"
    fi
    
    ./chain_replication -id $node_id -p $BASE_PORT -n $NUM_NODES > "server_$node_id.log" 2>&1 &
    echo $! >> pids.txt
    
    wait_for_port $port
    sleep 1
}

for i in $(seq $LAST_NODE -1 0); do
    start_node $i
done

sleep 2

echo "Starting client connected to head (port $BASE_PORT) and tail (port $((BASE_PORT + LAST_NODE)))..."
./chain_replication -client -p $BASE_PORT -n $NUM_NODES

cleanup() {
    echo
    echo "Cleaning up processes..."
    if [ -f pids.txt ]; then
        while read pid; do
            kill $pid 2>/dev/null
        done < pids.txt
        rm pids.txt
    fi
    exit 0
}

trap cleanup SIGINT SIGTERM EXIT

wait
