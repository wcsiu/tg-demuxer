build-image:
	docker build -f build/Dockerfile -t telegram/promoter .

run:
	LD_LIBRARY_PATH=/usr/local/lib/ go run ./cmd/run