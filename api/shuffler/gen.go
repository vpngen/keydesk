package shuffler

//go:generate mkdir -p ../../gen/shuffler
//go:generate oapi-codegen --config=types.yaml openapi.yaml
//go:generate oapi-codegen --config=spec.yaml openapi.yaml
//go:generate oapi-codegen --config=server.yaml openapi.yaml
//go:generate oapi-codegen --config=client.yaml openapi.yaml
