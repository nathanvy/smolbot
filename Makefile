APPNAME=smolbot

.PHONY: dev release clean

dev:
	go build -o $(APPNAME)

release:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $(APPNAME)-linux-amd64

clean:
	rm -f $(APPNAME) $(APPNAME)-linux-amd64
