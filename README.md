# Bocchi-Re

[![Go Report Card](https://goreportcard.com/badge/github.com/tpc3/vanilla)](https://goreportcard.com/report/github.com/tpc3/Bocchi-Re)
[![Docker Image CI](https://github.com/tpc3/Vanilla/actions/workflows/docker-image.yml/badge.svg)](https://github.com/tpc3/Bocchi-Re/actions/workflows/docker-image.yml)

Discord Bot can use AI made by OpenAI on Discord.

# 使用方法

## Simple

1. [Download config.template.yaml](https://raw.githubusercontent.com/tpc3/Vanilla/master/config.template.yaml)
2. Enter your discord token and OpenAI token to `config.template.yaml`
3. Change the name `config.template.yaml` to `config.yaml`
4. `go run main.go`

## Docker

1. [Download config.template.yaml](https://raw.githubusercontent.com/tpc3/Vanilla/master/config.template.yaml)
2. Enter your discord token and OpenAI token to `config.template.yaml`
3. Change the name `config.template.yaml` to `config.yaml`
3. `docker run --rm -it -v $(PWD):/data ghcr.io/tpc3/bocchi-re`

# Contribution

Contributions are always welcome. (Please make issue or PR with English or Japanese)