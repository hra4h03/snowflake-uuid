MODULE      := github.com/hra4h03/snowflake-uuid
BINARY      := snowflake-server
BUILD_DIR   := ./bin
DOCKER_TAG  := snowflake-uuid:latest

.PHONY: build test test-cover bench lint docker-build clean install-systemd

build:
	go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY) ./cmd/snowflake-server

test:
	go test ./... -race -count=1 -v

test-cover:
	go test ./... -race -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

bench:
	go test -bench=. -benchmem ./snowflake/

lint:
	go vet ./...

docker-build:
	docker build -f deploy/Dockerfile -t $(DOCKER_TAG) .

clean:
	rm -rf $(BUILD_DIR) coverage.out coverage.html

install-systemd: build
	sudo cp $(BUILD_DIR)/$(BINARY) /usr/local/bin/$(BINARY)
	sudo cp deploy/snowflake-uuid.service /etc/systemd/system/
	sudo mkdir -p /etc/snowflake-uuid
	sudo systemctl daemon-reload
	sudo systemctl enable snowflake-uuid
	@echo "Installed. Set SNOWFLAKE_NODE_ID in /etc/snowflake-uuid/snowflake-uuid.env"
	@echo "Then: sudo systemctl start snowflake-uuid"
