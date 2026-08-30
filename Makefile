.PHONY: up network-inspector-up front-up

PROFILE_RETRIEVAL_DIR := profile-retrieval-service
NETWORK_INSPECTOR_DIR := network-inspector
FRONT_APP_DIR := front-app

up:
	cd $(PROFILE_RETRIEVAL_DIR) && go run ./cmd

network-inspector-up:
	cd $(NETWORK_INSPECTOR_DIR) && node traffic_inspector.js

front-up:
	cd $(FRONT_APP_DIR) && npm start
