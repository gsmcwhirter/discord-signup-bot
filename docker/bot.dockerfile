ARG REPO
################################################################################
FROM $REPO/signup-base:latest-build AS build

WORKDIR /build
COPY go.mod go.sum ./
COPY bin ./bin/
COPY tools ./tools/
RUN bin/botctl go deps

COPY . .
RUN bin/botctl go gtl ./... && \
    bin/botctl go build ./cmd/db-migrate/... && \
    bin/botctl go build ./cmd/trials-bot/... && \
    bin/botctl go build ./cmd/trials-cleanup/... && \
    bin/botctl go build ./cmd/trials-dump/...

################################################################################
FROM $REPO/signup-base:latest-runtime as bot
WORKDIR /app

COPY --from=build --chown=discordbot:discordbot /build/out/trials-bot ./
COPY --chown=discordbot:discordbot bin/botlib bin/start-bot ./bin/
COPY --chown=discordbot:discordbot config.toml ./
COPY --chown=discordbot:discordbot honeytail.conf.tmpl ./

USER discordbot

ENTRYPOINT [ "/app/bin/start-bot" ]