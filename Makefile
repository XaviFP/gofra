# https://stackoverflow.com/a/73912048

build:
	go build -o bin/gofra . ;

build_plugins:
	go build -buildmode=plugin -o bin/plugins/cryptoasset_info.so plugins/cryptoasset_info/cryptoasset_info.go ;
	go build -buildmode=plugin -o bin/plugins/command.so plugins/command/command.go ;
	go build -buildmode=plugin -o bin/plugins/muc.so plugins/muc/muc.go ;
	go build -buildmode=plugin -o bin/plugins/reminder.so plugins/reminder/reminder.go ;
	go build -buildmode=plugin -o bin/plugins/pairs_price.so plugins/pairs_price/pairs_price.go ;
	go build -buildmode=plugin -o bin/plugins/dice.so plugins/dice/dice.go ;
	go build -buildmode=plugin -o bin/plugins/pick.so plugins/pick/pick.go ;
	go build -buildmode=plugin -o bin/plugins/trivia.so plugins/trivia/* ;
	go build -buildmode=plugin -o bin/plugins/session_tracker.so plugins/session_tracker/* ;
	go build -buildmode=plugin -o bin/plugins/list.so plugins/list/* ;
	go build -buildmode=plugin -o bin/plugins/help.so plugins/help/help.go ;
	go build -buildmode=plugin -o bin/plugins/web_title.so plugins/web_title/web_title.go ;

build_test_plugins:
	go build -buildmode=plugin -o test_plugins/bin/naughty.so test_plugins/naughty/naughty.go
	go build -buildmode=plugin -o test_plugins/bin/normie.so test_plugins/normie/normie.go
	go build -buildmode=plugin -o test_plugins/bin/not_really.so test_plugins/not_really/not_really.go

all: build build_plugins

test: build_test_plugins
	go test -p=1 -coverprofile=coverage.out *.go && go tool cover -html=coverage.out
