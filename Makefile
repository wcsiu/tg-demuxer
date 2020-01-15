build-image:
	docker build -f build/Dockerfile -t telegram/demuxer .

run:
	LD_LIBRARY_PATH=/usr/local/lib/ go run ./cmd/run

up:
	docker-compose -fbuild/docker-compose.yml up

db:
	docker-compose -fbuild/docker-compose.yml up demuxer-postgres

db-up:
	docker run -it --rm --net=build_default -v ${PWD}/db/migrations:/migrations migrate/migrate -path /migrations -database "postgres://postgres:postgres@demuxer-postgres:5432/demuxer-dev?sslmode=disable" up

db-down:
	docker run -it --rm --net=build_default -v "${PWD}"/db/migrations:/migrations migrate/migrate -path /migrations -database "postgres://postgres:postgres@demuxer-postgres:5432/demuxer-dev?sslmode=disable" down