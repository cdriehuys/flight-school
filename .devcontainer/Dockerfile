FROM mcr.microsoft.com/devcontainers/go:1-1.23-bookworm

# Postgresql client for debugging database
RUN apt-get update \
    && apt-get install -y postgresql-client

# Install `just`
RUN curl --proto '=https' --tlsv1.2 -sSf https://just.systems/install.sh | bash -s -- --to /usr/local/bin
