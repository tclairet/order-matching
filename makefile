export DB_URL := postgres://tranched:password@localhost:5432/tranched?sslmode=disable

build: clean
	@mkdir bin
	go build -o bin

test:
	docker-compose --file docker-compose-test.yml up -d
	@sleep 2
	go test ./... -race -v -timeout 20s
	@make clean-docker

clean-docker:
	docker-compose --file docker-compose-test.yml stop

clean:
	@rm -rf bin