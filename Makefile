PATH := $(shell pwd)/sbin:$(PATH)

ifeq ($(strip $(LOCAL)),)
    IN_CONTAINER := container-run
endif

BUILD := $(IN_CONTAINER) build-for-arch
DEBUILD := debuild -i -us -uc -b
DEBUILD_CLEAN := debuild -- clean
LINTIAN_OPTIONS = --lintian-opts --profile debian --suppress-tags statically-linked-binary

ARCHITECTURES := amd64 arm64

.PHONY: build test deb debs all $(ARCHITECTURES)

# Default
build:
	$(BUILD)

test:
	$(IN_CONTAINER) go test ./...

deb: build
	$(DEBUILD) $(LINTIAN_OPTIONS)
	$(DEBUILD_CLEAN)

all:
	$(BUILD) $(ARCHITECTURES)

$(ARCHITECTURES): all
	export DEB_BUILD_ARCH=$@; $(DEBUILD) -a $@ $(LINTIAN_OPTIONS)
	$(DEBUILD_CLEAN)

debs: all $(ARCHITECTURES)
