GOCACHE ?= /tmp/go-build-cache
COVERAGE_MIN ?= 60.0

.PHONY: build test coverage run-demo score-demo gate-demo

build:
	env GOCACHE=$(GOCACHE) go build -o seh .

test:
	env GOCACHE=$(GOCACHE) go test -race ./...

coverage:
	env GOCACHE=$(GOCACHE) go test ./... -coverprofile=coverage.out
	env GOCACHE=$(GOCACHE) go tool cover -func=coverage.out
	@total=$$(env GOCACHE=$(GOCACHE) go tool cover -func=coverage.out | awk '/^total:/ {print $$3}' | tr -d '%'); \
	awk 'BEGIN { if ('"$$total"' + 0 < $(COVERAGE_MIN)) exit 1 }' || \
	( echo "coverage $$total% is below required $(COVERAGE_MIN)%"; exit 1 )

run-demo: build
	mkdir -p demo/out
	./seh run --skill demo-skill --cases demo/cases --out demo/out/run.json

score-demo: run-demo
	mkdir -p demo/out
	./seh score --run demo/out/run.json --out demo/out/score.json

gate-demo: score-demo
	./seh gate --report demo/out/score.json --policy demo/policy.yaml
