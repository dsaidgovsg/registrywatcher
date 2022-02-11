module github.com/dsaidgovsg/registrywatcher

require (
	github.com/docker/distribution v2.8.0+incompatible // indirect
	github.com/docker/docker v17.12.0-ce-rc1.0.20200330121334-7f8b4b621b5d+incompatible
	github.com/docker/go-connections v0.4.0
	github.com/fatih/color v1.12.0 // indirect
	github.com/gin-contrib/cors v1.3.0
	github.com/gin-gonic/gin v1.4.0
	github.com/gorilla/mux v1.7.4
	github.com/hashicorp/nomad v1.1.3
	github.com/hashicorp/nomad/api v0.0.0-20200529203653-c4416b26d3eb
	github.com/jmoiron/sqlx v1.2.0
	github.com/lib/pq v1.1.0
	github.com/lusis/go-slackbot v0.0.0-20210303200821-3c34a039d473 // indirect
	github.com/lusis/slack-test v0.0.0-20190426140909-c40012f20018 // indirect
	github.com/mattn/go-isatty v0.0.13 // indirect
	github.com/nlopes/slack v0.5.0
	github.com/pkg/errors v0.9.1
	github.com/spf13/viper v1.3.2
	github.com/stretchr/testify v1.7.0
	go.uber.org/zap v1.10.0
	golang.org/x/crypto v0.0.0-20210711020723-a769d52b0f97 // indirect
	golang.org/x/net v0.0.0-20210405180319-a5a99cb37ef4 // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c // indirect
	golang.org/x/sys v0.0.0-20210630005230-0f9fa26af87c // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	gotest.tools/v3 v3.0.3 // indirect
)

replace github.com/ugorji/go v1.1.4 => github.com/ugorji/go/codec v0.0.0-20190204201341-e444a5086c43

go 1.13
