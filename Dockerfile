FROM golang:1.27.3-trixie AS build

RUN apt-get update \
 && apt-get install -y --no-install-recommends gzip \
 && rm -rf /var/lib/apt/lists/*

WORKDIR /src

FROM build AS dev

RUN apt-get update \
 && apt-get install -y --no-install-recommends ffmpeg \
 && rm -rf /var/lib/apt/lists/*
