package swagger

//go:generate go run github.com/go-swagger/go-swagger/cmd/swagger@latest generate server -t ../gen -f swagger.yml --exclude-main -A user
//go:generate go run github.com/go-swagger/go-swagger/cmd/swagger@latest generate client -t ../gen -f swagger.yml
