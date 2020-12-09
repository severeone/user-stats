package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/fluxio/multierror"
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

const DbDriverName = "mysql"

type StorageService interface {
	// Gracefully end the service
	Close() error

	// SetCid stores a client ID with a corresponding date
	SetCid(cid uuid.UUID, date time.Time) error

	// GetUniqueDailyCidsCount retrieves the number of unique client IDs
	// for the given date
	GetUniqueDailyCidCount(date time.Time) (int, error)

	// GetUniqueMontlyCidsCount retrieves the number of unique client IDs
	// for the month prior to and including the given date
	GetUniqueMonthlyCidCount(date time.Time) (int, error)
}

type cidsEntry struct {
	Cid  string `db:"cid"`
	Date string `db:"date"`
}

type dateRangeEntry struct {
	StartDate string `db:"start_date"`
	EndDate   string `db:"end_date"`
}

// dbStorageService wraps the clients database and database connection information
type dbStorageService struct {
	db                        *sqlx.DB
	sql_SaveCid               *sqlx.NamedStmt
	sql_UniqueDailyCidCount   *sqlx.NamedStmt
	sql_UniqueMonthlyCidCount *sqlx.NamedStmt
}

// NewDbStorageService creates a new connection to MySQL database
func NewDbStorageService(config *Config) (StorageService, error) {
	dbSourceName := fmt.Sprintf("%s:%s@(%s:%s)/%s",
		config.Mysql.Username,
		config.Mysql.Password,
		config.Mysql.Host,
		config.Mysql.Port,
		config.Mysql.Name,
	)
	db, err := sqlx.Connect(DbDriverName, dbSourceName)
	if err != nil {
		log.Fatalf("Failed to connect to db %q: %v", dbSourceName, err)
		return nil, err
	}
	db.SetMaxOpenConns(config.Mysql.MaxConnections)

	var errs multierror.Accumulator
	prepareNamed := func(statement string) *sqlx.NamedStmt {
		s, e := db.PrepareNamed(statement)
		errs.Push(e)
		return s
	}

	s := &dbStorageService{
		db: db,
		sql_SaveCid: prepareNamed(`
				INSERT IGNORE INTO cids (cid, date) VALUES (:cid, :date)`),
		sql_UniqueDailyCidCount: prepareNamed(`
				SELECT COUNT(DISTINCT cid) FROM cids WHERE date=:date`),
		sql_UniqueMonthlyCidCount: prepareNamed(`
				SELECT COUNT(DISTINCT cid) FROM cids WHERE date>=:start_date AND date<=:end_date`),
	}

	if errs.Error() != nil {
		errs.Push(s.Close())
		return nil, errs.Error()
	}

	return s, nil
}

func (s *dbStorageService) Close() error {
	return s.db.Close()
}

func (s *dbStorageService) SetCid(cid uuid.UUID, date time.Time) error {
	dateStr := date.Format("20060102")
	_, err := s.sql_SaveCid.Exec(&cidsEntry{
		Cid:  cid.String(),
		Date: dateStr,
	})
	if err != nil {
		return fmt.Errorf("Failed to save a client ID %q %q: %v", cid.String(), dateStr, err)
	}
	return nil
}

func (s *dbStorageService) GetUniqueDailyCidCount(date time.Time) (int, error) {
	var count int
	d := date.Format("20060102")
	err := s.sql_UniqueDailyCidCount.Get(&count, cidsEntry{Date: d})
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, fmt.Errorf("Failed to execute a query by date %q: %v", d, err)
	}
	return count, nil
}

func (s *dbStorageService) GetUniqueMonthlyCidCount(date time.Time) (int, error) {
	var count int
	startDate := date.AddDate(0, -1, 0).Format("20060102")
	endDate := date.Format("20060102")
	err := s.sql_UniqueMonthlyCidCount.Get(&count, dateRangeEntry{
		StartDate: startDate,
		EndDate:   endDate,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, fmt.Errorf(
			"Failed to execute a query by date range %q - %q: %v",
			startDate, endDate, err)
	}
	return count, nil
}
