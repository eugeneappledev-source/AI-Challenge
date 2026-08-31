.PHONY: backend-test backend-run ios-generate ios-build

backend-test:
	cd backend && go test ./... && go vet ./...

backend-run:
	cd backend && set -a && . ../.env && set +a && go run ./cmd/api

ios-generate:
	cd ios && xcodegen generate

ios-build: ios-generate
	cd ios && xcodebuild -project AIChallenge.xcodeproj -scheme AIChallenge -destination 'generic/platform=iOS Simulator' -derivedDataPath /tmp/ai-challenge-derived-data CODE_SIGNING_ALLOWED=NO build
