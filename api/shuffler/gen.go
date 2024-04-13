package messages

//go:generate mkdir -p ../../gen/shuffler
//go:generate go run github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen@v2.1 --config=types.yaml openapi.yaml
//go:generate go run github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen@v2.1 --config=spec.yaml openapi.yaml
//go:generate go run github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen@v2.1 --config=server.yaml openapi.yaml
//go:generate go run github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen@v2.1 --config=client.yaml openapi.yaml
