build-image:
	docker build -f build/Dockerfile -t telegram/promoter .

build-binary:
	go build ./cmd/run