#!/usr/bin/env bash

: ${BOT_CONFIG:=./trials-bot-test-config.toml}
: ${DB_FILE:=./test.db}
: ${HONEYTAIL_CONFIG:=./honeytail.conf}
: ${NUM_WORKERS:=4}
: ${BOT_BINARY:=./bin/trials-bot}

export BOT_CONFIG="$BOT_CONFIG" 
export DB_FILE="$DB_FILE" 
export HONEYTAIL_CONFIG="$HONEYTAIL_CONFIG"
export NUM_WORKERS="$NUM_WORKERS"
export BOT_BINARY="$BOT_BINARY"
exec ./start-bot.sh