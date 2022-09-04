package main

import (
	"log"

	"github.com/go-openapi/loads"

	"github.com/vpngen/keykeeper/gen/restapi"
	"github.com/vpngen/keykeeper/gen/restapi/operations"
	"github.com/vpngen/keykeeper/token"
)

//go:generate swagger generate server -t ../../gen -f ../../swagger/swagger.yml --exclude-main -A user

// BrigadierID - Brigadier ID from the env.
var BrigadierID = "fjsdjfsdjf"

// TokenLifeTime - token time to life.
const TokenLifeTime = 3600

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

	api.BearerAuth = token.ValidateBearer(BrigadierID)
	api.PostTokenHandler = operations.PostTokenHandlerFunc(token.CreateToken(BrigadierID, TokenLifeTime))

	server.ConfigureAPI()

	// serve API
	if err := server.Serve(); err != nil {
		log.Fatalln(err)
	}
}
