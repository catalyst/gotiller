PATH := $(shell pwd)/sbin:$(PATH)

ifeq ($(strip $(LOCAL)),)
    IN_CONTAINER := container-run
endif

.PHONY: all test

all:
	$(IN_CONTAINER) build-for-arch $(ARCH)

test:
	$(IN_CONTAINER) go test ./...

