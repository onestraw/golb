build:
	go build -o server

prepare:
	mkdir -p consul.d

run: build prepare
	goreman start

clean:
	rm -rf server consul.d
