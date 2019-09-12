build-image:
	docker build -f build/Dockerfile -t telegram/demuxer .

run:
	LD_LIBRARY_PATH=/usr/local/lib/ go run ./cmd/run

up:
	docker-compose -fbuild/docker-compose.yml up