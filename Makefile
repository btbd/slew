slew.exe: *.go
	go build -ldflags "-w -extldflags -static" -tags netgo -installsuffix netgo -o slew.exe

clean:
	rm -f slew.exe