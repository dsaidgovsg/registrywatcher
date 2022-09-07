module github.com/dsaidgovsg/registrywatcher

require (
	github.com/docker/docker v20.10.17+incompatible
	github.com/docker/go-connections v0.4.0
	github.com/gin-contrib/cors v1.4.0
	github.com/gin-gonic/gin v1.8.1
	github.com/go-test/deep v1.0.4 // indirect
	github.com/gorilla/mux v1.8.0
	github.com/hashicorp/nomad v1.3.5
	github.com/hashicorp/nomad/api v0.0.0-20220707195938-75f4c2237b28
	github.com/jmoiron/sqlx v1.3.5
	github.com/lib/pq v1.10.7
	github.com/nlopes/slack v0.6.0
	github.com/pkg/errors v0.9.1
	github.com/spf13/viper v1.12.0
	github.com/stretchr/testify v1.8.0
	go.uber.org/zap v1.21.0
	golang.org/x/net v0.0.0-20220624214902-1bab6f366d9e // indirect
	golang.org/x/sync v0.0.0-20220601150217-0de741cfad7f // indirect
	golang.org/x/sys v0.0.0-20220624220833-87e55d714810 // indirect
	golang.org/x/xerrors v0.0.0-20220609144429-65e65417b02f // indirect
	google.golang.org/genproto v0.0.0-20220815135757-37a418bb8959 // indirect
)

replace github.com/ugorji/go v1.1.4 => github.com/ugorji/go/codec v0.0.0-20190204201341-e444a5086c43

go 1.13
