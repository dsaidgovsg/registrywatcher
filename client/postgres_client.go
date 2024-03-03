package client

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type PostgresClient struct {
	db   *sqlx.DB
	conf *viper.Viper
}

// Initialize creates tables if they do not exist
// cloud agnostic function
func InitializePostgresClient(conf *viper.Viper) (*gorm.DB, error) {
	// client := PostgresClient{
	// 	conf: conf,
	// }
	// var dburl string
	// if len(os.Getenv("DATABASE_URL")) > 0 {
	// 	dburl = os.Getenv("DATABASE_URL")
	// } else {
	// 	dburl = conf.GetString("database_url")
	// }
	// createSchema := conf.GetBool("create_database_schema")

	// var err error
	// if client.db, err = sqlx.Open("postgres", dburl); err != nil {
	// 	return &client, errors.Wrap(err, "unable to open postgres db")
	// }

	// if createSchema {
	// 	// Since this happens at initialization we
	// 	// could encounter racy conditions waiting for pg
	// 	// to become available. Wait for it a bit
	// 	if err = client.db.Ping(); err != nil {
	// 		// Try 3 more times
	// 		// 5, 10, 20
	// 		for i := 0; i < 3 && err != nil; i++ {
	// 			time.Sleep(time.Duration(5*math.Pow(2, float64(i))) * time.Second)
	// 			err = client.db.Ping()
	// 		}
	// 		if err != nil {
	// 			return &client, errors.Wrap(err, "error trying to connect to postgres db, retries exhausted")
	// 		}
	// 	}

	// 	if err = client.createTables(); err != nil {
	// 		return &client, errors.Wrap(err, "problem executing create tables sql")
	// 	}
	// 	if err = client.initRows(); err != nil {
	// 		return &client, errors.Wrap(err, "problem executing init row sql")
	// 	}
	// }
	dsn := "host=localhost user=gorm password=gorm dbname=gorm port=9920 sslmode=disable TimeZone=Asia/Shanghai"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}

func (client *PostgresClient) createTables() error {
	_, err := client.db.Exec(CreateTablesSQL)
	return err
}

func (client *PostgresClient) initRows() error {
	for _, repoName := range client.conf.GetStringSlice("watched_repositories") {
		tx, err := client.db.Begin()
		if err != nil {
			return errors.WithStack(err)
		}

		if _, err = tx.Exec(InsertRowSql,
			repoName, "", true); err != nil {
			tx.Rollback()
			return errors.Wrapf(err, "issue populating new watched repo with default tag empty string [%s]", repoName)
		}

		if err = tx.Commit(); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

func (client *PostgresClient) UpdateAutoDeployFlag(repoName string, autoDeploy bool) error {
	tx, err := client.db.Begin()
	if err != nil {
		return errors.WithStack(err)
	}

	update := `
          UPDATE deployed_repository_version SET auto_deploy = $2 WHERE repository_name = $1;`

	if _, err = tx.Exec(
		update, repoName, autoDeploy); err != nil {
		tx.Rollback()
		return errors.WithStack(err)
	}

	if err = tx.Commit(); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (client *PostgresClient) UpdatePinnedTag(repoName, pinnedTag string) error {
	tx, err := client.db.Begin()
	if err != nil {
		return errors.WithStack(err)
	}

	update := `
          UPDATE deployed_repository_version SET pinned_tag = $2 WHERE repository_name = $1;`

	if _, err = tx.Exec(
		update, repoName, pinnedTag); err != nil {
		tx.Rollback()
		return errors.WithStack(err)
	}

	if err = tx.Commit(); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

type DeployedRepositoryVersionRow struct {
	PinnedTag      string `json:"pinned_tag" db:"pinned_tag"`
	RepositoryName string `json:"repository_name" db:"repository_name"`
	AutoDeploy     bool   `json:"auto_deploy" db:"auto_deploy"`
}

func (client *PostgresClient) GetAutoDeployFlag(repoName string) (bool, error) {
	var rtn DeployedRepositoryVersionRow
	sqlStatement := "select * from deployed_repository_version where repository_name = $1"
	err := client.db.Get(&rtn, sqlStatement, repoName)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, errors.Wrapf(err, "auto_deploy with repoName %s not found", repoName)
		} else {
			return false, errors.Wrapf(err, "issue getting auto_deploy with repoName [%s]", repoName)
		}
	}
	return rtn.AutoDeploy, err
}

func (client *PostgresClient) GetPinnedTag(repoName string) (string, error) {
	var rtn DeployedRepositoryVersionRow
	sqlStatement := "select * from deployed_repository_version where repository_name = $1"
	err := client.db.Get(&rtn, sqlStatement, repoName)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", errors.Wrapf(err, "pinned_tag with repoName %s not found", repoName)
		} else {
			return "", errors.Wrapf(err, "issue getting pinned_tag with repoName [%s]", repoName)
		}
	}
	return rtn.PinnedTag, err
}

func (client *PostgresClient) GetAllTags() (map[string]string, error) {
	var rows []DeployedRepositoryVersionRow
	sqlStatement := "select * from deployed_repository_version"

	err := client.db.Select(&rows, sqlStatement)
	rtn := make(map[string]string)
	for _, row := range rows {
		rtn[row.RepositoryName] = row.PinnedTag
	}
	return rtn, err
}

const InsertRowSql = `
INSERT INTO deployed_repository_version
  (repository_name, pinned_tag, auto_deploy) VALUES ($1, $2, $3)
  ON CONFLICT (repository_name) DO NOTHING;`

const CreateTablesSQL = `
CREATE TABLE IF NOT EXISTS deployed_repository_version (
  repository_name character varying NOT NULL PRIMARY KEY UNIQUE,
  pinned_tag character varying NOT NULL,
  auto_deploy boolean NOT NULL default true
);`
