# DLV-PLAT-004 User Story 3 — Go API Artifact Verification

## Scope

This record verifies the OpenAPI-first Go artifact workflow only. It does not
claim TypeScript generation, focused CI drift enforcement, finance handlers, or
finance behavior.

## Tool and source contract

- Go: `1.26.3`
- ogen: `v1.23.0`
- OpenAPI source: `contracts/openapi/openapi.yaml`
- Generated package: `internal/platform/httpapi/generated/`
- Generated package name: `generated`
- Expected artifact inventory: `scripts/openapi/expected-go-artifacts.txt`
- Configuration checksum command: `sha256sum .ogen.yml`
- Configuration checksum: `c0b812578b1b34528025e9340d160adc176a51d410e7781a88f897562c8194ce`

The generation wrapper bundles the multi-file OpenAPI source with Redocly into
a temporary file before invoking ogen. This is required because the committed
contract uses an external `paths` reference.

## Expected generated artifacts

The normalized inventory is maintained at
`scripts/openapi/expected-go-artifacts.txt` and currently contains:

```text
oas_cfg_gen.go
oas_handlers_gen.go
oas_interfaces_gen.go
oas_json_gen.go
oas_middleware_gen.go
oas_operations_gen.go
oas_parameters_gen.go
oas_request_decoders_gen.go
oas_response_encoders_gen.go
oas_router_gen.go
oas_schemas_gen.go
oas_server_gen.go
oas_unimplemented_gen.go
oas_validators_gen.go
```

No `oas_client_gen.go` artifact is generated.

## Commands executed

Commands were run from the repository root. The sandbox required a writable
temporary Go build cache, so verification commands used `GOCACHE=/tmp/tally-go-cache`.

1. `go get -tool github.com/ogen-go/ogen/cmd/ogen@v1.23.0`
2. `GOCACHE=/tmp/tally-go-cache go mod tidy`
3. `go tool ogen -version`
4. `GOCACHE=/tmp/tally-go-cache make api-generate`
5. `GOCACHE=/tmp/tally-go-cache make api-generate-check`
6. `GOCACHE=/tmp/tally-go-cache make api-negative-check`

## Results

- ogen resolved to `v1.23.0`.
- OpenAPI bundling completed successfully.
- Go generation completed successfully.
- Repeated generation was deterministic.
- Generated markers were verified.
- No Go client artifact was generated.
- `go test ./...` passed through `make api-generate-check`.
- `go build ./cmd/api` passed through `make api-generate-check`.
- Negative checks rejected manual edits, missing files, extra files, and invalid input with non-zero status.
- Negative checks completed in temporary directories and did not modify the working tree.
