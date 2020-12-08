package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"gopkg.in/yaml.v2"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

const (
	CidParam  = "cid"
	DateParam = "d"

	CollectCidsPath       = "/collect"
	DailyUniqueCidsPath   = "/daily_uniques"
	MonthlyUniqueCidsPath = "monthly_uniques"

	ContentTypeForm = "application/x-www-form-urlencoded"

	ConfigFileFlag = "config"
	ConfigFilePath = "./config.yml"

	DbDriverName = "mysql"
)

// Config struct for app configuration file
type Config struct {
	Mysql struct {
		Host string `yaml:"host"`
		Port string `yaml:"port"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		Name string `yaml:"name"`
	} `yaml:"mysql"`
	Server struct {
		Host string `yaml:"host"`
		Port string `yaml:"port"`
	} `yaml:"server"`
}

func main() {
	// Parse command line flags
	configPath, err := parseFlags()
	if err != nil {
		log.Fatal(err)
	}

	// Read configuration
	config, err := newConfig(configPath)
	if err != nil {
		log.Fatal(err)
	}

	// Create etcd client
	db, err := newDbConnection(config)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create router and start a server
	r := newRouter(db)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%s", config.Server.Host, config.Server.Port), r))
}

// parseFlags will create and parse the CLI flags and return the path to be used elsewhere
func parseFlags() (string, error) {
	var configPath string
	flag.StringVar(&configPath, ConfigFileFlag, ConfigFilePath, "path to config file")
	flag.Parse()
	if err := validateConfigPath(configPath); err != nil {
		return "", err
	}
	return configPath, nil
}

// validateConfigPath just makes sure that the path can be read
func validateConfigPath(path string) error {
	s, err := os.Stat(path)
	if err != nil {
		return err
	}
	if s.IsDir() {
		return fmt.Errorf("'%s' is a directory, not a normal file", path)
	}
	return nil
}

// newConfig returns a new decoded Config struct
func newConfig(configPath string) (*Config, error) {
	config := &Config{}

	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	d := yaml.NewDecoder(file)
	if err := d.Decode(&config); err != nil {
		return nil, err
	}

	return config, nil
}

// newDbConnection creates a new connection to MySQL database
func newDbConnection(config *Config) (*sql.DB, error) {
	return sql.Open(DbDriverName, fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",
			config.Mysql.Username,
			config.Mysql.Password,
			config.Mysql.Host,
			config.Mysql.Port,
			config.Mysql.Name,
		))
}

func newRouter(db *sql.DB) *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc(CollectCidsPath, collectCid(db)).Methods(http.MethodGet)
	r.HandleFunc(DailyUniqueCidsPath, dailyUniqueCids(db)).Methods(http.MethodGet)
	r.HandleFunc(MonthlyUniqueCidsPath, monthlyUniqueCids(db)).Methods(http.MethodGet)

	return r
}

// collectCid registers a single client ID
func collectCid(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", ContentTypeForm)
		query := r.URL.Query()

		// Parse client ID
		cid, err := uuid.Parse(query.Get(CidParam))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Parse date
		tm := time.Now().UTC()
		date := query.Get(DateParam)
		if date != "" {
			i, err := strconv.ParseInt(date, 10, 64)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			tm = time.Unix(i, 0)
		}

		// Save client ID to the storage
		saveCid(db, tm, cid)

		w.WriteHeader(http.StatusOK)
	}
}

// saveCid puts a client ID with a date to the storage
func saveCid(db *sql.DB, tm time.Time, cid uuid.UUID) {
	key := fmt.Sprintf("%s%s", tm.Format("20200909"), cid.String())
	_, err := kv.Put(context.TODO(), key, "1")
	if err != nil {
		log.Printf("Failed to save a client id: %s", cid.String())
	}
}

func dailyUniqueCids(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", ContentTypeForm)
		query := r.URL.Query()

		// Parse date
		date := query.Get(DateParam)
		tm, err := time.Parse("20200909", date)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}


	}
}