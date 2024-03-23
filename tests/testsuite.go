package testutils

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"registrywatcher/app/models"
	"registrywatcher/app/repository"
	"registrywatcher/server"
	"registrywatcher/utilities/config"
	"registrywatcher/utilities/db"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type TestSuite struct {
	suite.Suite
	r   *gin.Engine
	db  *gorm.DB
	Txn *gorm.DB

	// User configurable test parameters, must be reset after each subtest
	Data              *DataSeed
	Config            *config.Config
	RepositoryService *repository.RepositoryService
}

func (suite *TestSuite) SetupSuite() {
	c := config.InitConfig(config.GetEnv())
	suite.db = db.InitDB(c.DSN)
}

func (suite *TestSuite) SetupSubTest() {
	if suite.Config == nil {
		suite.Config = config.InitConfig(config.Testing)
	}

	suite.Txn = suite.db.Begin()
	suite.Data.Seed(suite.Txn)

	if suite.RepositoryService == nil {
		suite.RepositoryService = &repository.RepositoryService{
			DB:     suite.Txn,
			Config: suite.Config,
		}
	}

	suite.r = server.InitRoutes(*suite.RepositoryService)
}

func (suite *TestSuite) TearDownSubTest() {
	suite.Config = nil
	suite.Txn.Rollback()
	suite.RepositoryService = nil
}

func (suite *TestSuite) TearDownSuite() {
	if err := suite.db.Migrator().DropTable(&models.Repository{}); err != nil {
		suite.T().Fatal(err)
	}
}

func (suite *TestSuite) SendRequest(method string, path string, body interface{},
	headers map[string]string) *httptest.ResponseRecorder {

	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(body)
	if err != nil {
		suite.T().Fatal(err)
	}
	req, _ := http.NewRequest(method, path, &buf)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	suite.r.ServeHTTP(w, req)

	return w
}
