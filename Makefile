build:
	go build -o bin/gofra .

build_plugins:
	go build -buildmode=plugin -o bin/plugins/cryptoasset_info.so plugins/cryptoasset_info.go
	go build -buildmode=plugin -o bin/plugins/command.so plugins/command.go
	#go build -buildmode=plugin -o bin/plugins/muc.so plugins/muc.go
	go build -buildmode=plugin -o bin/plugins/reminder.so plugins/reminder.go
	go build -buildmode=plugin -o bin/plugins/pairs_price.so plugins/pairs_price.go

build_test_plugins:
	go build -buildmode=plugin -o test_plugins/bin/naughty.so test_plugins/naughty.go
	go build -buildmode=plugin -o test_plugins/bin/normie.so test_plugins/normie.go
	go build -buildmode=plugin -o test_plugins/bin/not_really.so test_plugins/not_really.go

all: build build_plugins

test: build build_test_plugins
