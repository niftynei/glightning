BUILD_DIR=build
TEST_BUILD_DIR=$(BUILD_DIR)/test
TEST_PLUGINS_BUILD_DIR=$(TEST_BUILD_DIR)/plugins
LN_PATH=/usr/local/bin/lightningd

PLUGINS_DIR=examples/plugin

TEST_PLUGIN_DIRS = $(wildcard examples/plugin/pl_*)
PLUGINS = $(foreach p,$(TEST_PLUGIN_DIRS),$p/$(p:$(PLUGINS_DIR)/pl_%=%).go)

all: build test-build

build:
	go build github.com/niftynei/glightning/glightning
	go build github.com/niftynei/glightning/gbitcoin
	go build github.com/niftynei/glightning/jrpc2

test-build: $(PLUGINS)
	@rm -rf $(TEST_PLUGINS_BUILD_DIR)
	@mkdir -p $(TEST_PLUGINS_BUILD_DIR)
	@$(foreach p,$(TEST_PLUGIN_DIRS), cd $p && go build -o $(CURDIR)/$(TEST_PLUGINS_BUILD_DIR)/$(p:$(PLUGINS_DIR)/pl_%=plugin_%) && cd -;)


check: check-all

check-all: test-build
	LIGHTNINGD_PATH=$(LN_PATH) PLUGINS_PATH=$(CURDIR)/$(TEST_PLUGINS_BUILD_DIR) go test -v ./...

check-units:
	go test -v -short ./...
