fmt:
	@echo "==> Fixing source code with gofmt..."
	gofmt -s -w ./$(PKG_NAME)

mockgen:
	go get github.com/golang/mock/gomock
	go install github.com/golang/mock/mockgen

	mockgen -source hosts/hosts.go -destination hosts/hosts_mock.go -package hosts -self_package github.com/julienduchesne/pull-request-reminder/hosts
	mockgen -source messages/messages.go -destination messages/messages_mock.go -package messages -self_package github.com/julienduchesne/pull-request-reminder/messages