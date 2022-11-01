module github.com/dsaidgovsg/registrywatcher

require (
	github.com/docker/docker v20.10.21+incompatible
	github.com/docker/go-connections v0.4.0
	github.com/gin-contrib/cors v1.4.0
	github.com/gin-gonic/gin v1.8.1
	github.com/go-test/deep v1.0.4 // indirect
	github.com/gorilla/mux v1.8.0
	github.com/hashicorp/nomad v1.2.6
	github.com/hashicorp/nomad/api v0.0.0-20200529203653-c4416b26d3eb
	github.com/jmoiron/sqlx v1.3.5
	github.com/lib/pq v1.10.7
	github.com/moby/term v0.0.0-20210619224110-3f7ff695adc6 // indirect
	github.com/nlopes/slack v0.6.0
	github.com/pkg/errors v0.9.1
	github.com/spf13/viper v1.12.0
	github.com/stretchr/testify v1.8.0
	go.uber.org/zap v1.21.0
	gotest.tools/v3 v3.0.3 // indirect
)

replace github.com/ugorji/go v1.1.4 => github.com/ugorji/go/codec v0.0.0-20190204201341-e444a5086c43

go 1.13
