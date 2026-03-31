all: harvest-cli conversation-extract

harvest-cli: harvest-cli-src/*.go
	cd harvest-cli-src && go build -o ../harvest-cli .

conversation-extract: conversation-extract-src/*.go
	cd conversation-extract-src && go build -o ../conversation-extract .

clean:
	rm -f harvest-cli conversation-extract

.PHONY: all clean
