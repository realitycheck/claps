# Claps Makefile
GO ?= go

.SHELLFLAGS = -c # Run commands in a -c flag
.PHONY: build install clean test generate all help
.DEFAULT: help

help: ## Help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
