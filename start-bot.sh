#!/usr/bin/env bash

: ${BOT_CONFIG:=/home/discordbot/eso-discord/trials-bot-config.toml}
: ${DB_FILE:=/home/discordbot/eso-discord/trialsbot.db}
: ${HONEYTAIL_CONFIG:=/etc/honeytail/honeytail.conf}
: ${NUM_WORKERS:=64}
: ${BOT_BINARY:=/home/discordbot/eso-discord/trials-bot}

export GOMAXPROCS=32

env
exec &> >(tee >(honeytail -c $HONEYTAIL_CONFIG))
exec $BOT_BINARY \
    --config $BOT_CONFIG \
    --database $DB_FILE \
    --num_workers $NUM_WORKERS