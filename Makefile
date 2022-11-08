image := docker-hello-go

build-debug:
	DOCKER_BUILDKIT=0 docker build -t $(image) . --no-cache

build:
	docker build -t $(image) .

run:
	docker run --name hello-go --rm --network my-net -p 8002:8001 -e REDIS=redis:6379 $(image)

run-log:
	docker run --name hello-go --rm --network my-net -p 8002:8001 \
		-v $(PWD)/logs:/app/logs \
		-e REDIS=redis:6379 \
		-e LOGS=./logs \
		$(image)

run-volume:
	docker run --name hello-go --rm --network my-net -p 8002:8001 \
		-v my-logs:/app/logs \
		-e REDIS=redis:6379 \
		-e LOGS=./logs \
		$(image)