package main

import (
	"log"

	"github.com/go-openapi/loads"

	"test/gen/restapi"
	"test/gen/restapi/operations"
)

func main() {
	// load embedded swagger file
	swaggerSpec, err := loads.Analyzed(restapi.SwaggerJSON, "")
	if err != nil {
		log.Fatalln(err)
	}

	// create new service API
	api := operations.NewUserAPI(swaggerSpec)
	server := restapi.NewServer(api)
	defer server.Shutdown()

	server.Port = 80

	// TODO: Set Handle

	server.ConfigureAPI()

	// serve API
	if err := server.Serve(); err != nil {
		log.Fatalln(err)
	}
}
