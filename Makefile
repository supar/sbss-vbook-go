.PHONY: build

UNAME_S=$(shell uname -s)

NAME = sbss-vbook
VERSION = $(shell cat VERSION | sed -e 's,\-.*,,')
RELEASE = $(shell cat VERSION | sed -e 's,.*\-,,')


BUILD_DIR = $(notdir $(shell pwd))
BUILD_DATE = $(shell date +%Y%m%d%H%M%S)
BUILD_ARCH = amd64

ifeq (${UNAME_S}, Darwin)
BUILD_SYSTEM = darwin
else
BUILD_SYSTEM = linux
endif

LDFLAGS = -ldflags "-X main.NAME=${NAME} -X main.VERSION=${VERSION} -X main.BUILDDATE=${BUILD_DATE}"

SOURCE_FILES = $(shell ls -AB | grep -i 'version$$\|makefile$$\|\.go$$')

# Debian build root
DEB_DIR = $(shell pwd)/build/debian
DEB_ROOT = $(DEB_DIR)/$(NAME)-$(VERSION)/debian
DEB_CONF = $(DEB_ROOT)/$(NAME).cfg
DEB_INITD = $(DEB_ROOT)/$(NAME).service
DEB_SOURCE = $(DEB_DIR)/$(NAME)_$(VERSION).orig.tar.gz
DEB_PKG = $(DEB_DIR)/$(NAME)_$(VERSION)-$(RELEASE)_$(BUILD_ARCH).deb

build:
	go build -o ./$(NAME) -v $(LDFLAGS)

test:
	@go test -v ./...

deb: .clean_deb $(DEB_PKG)

$(DEB_ROOT): contrib/debian
	mkdir -p $(DEB_ROOT)
	cp -ad $</* $@/
	find $@ -type f -exec sed -i -e"s/@VERSION@/$(VERSION)/g" {} \;

$(DEB_SOURCE): $(SOURCE_FILES)
	mkdir -p $(@D)
	tar --transform "s,^,$(NAME)-$(VERSION)/src/$(NAME)/," -f $@ -cz $^

$(DEB_CONF): contrib/conf/$(NAME).cfg
	mkdir -p $(@D)
	cp -ad $< $@

$(DEB_INITD): contrib/systemd/$(NAME).service
	mkdir -p $(@D)
	cp -ad $< $@

$(DEB_PKG): $(DEB_ROOT) $(DEB_SOURCE) $(DEB_CONF) $(DEB_INITD)
	cd $(DEB_DIR)/$(NAME)-$(VERSION) && \
	debuild --set-envvar BUILD_APP_VERSION=$(VERSION) -us -uc -b

.clean_deb:
	@rm -rf $(shell find . -type d -path "*build/debian*" -print -quit)

