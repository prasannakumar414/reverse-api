.PHONY: up network-inspector-up

PROFILE_RETRIEVAL_DIR := profile-retrieval-service
NETWORK_INSPECTOR_DIR := network-inspector

up:
	cd $(PROFILE_RETRIEVAL_DIR) && go run ./cmd

network-inspector-up:
	cd $(NETWORK_INSPECTOR_DIR) && node traffic_inspector.js
