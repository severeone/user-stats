package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-yaml/yaml"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

const (
	CidParam  = "cid"
	DateParam = "d"

	CollectCidsPath       = "/collect"
	DailyUniqueCidsPath   = "/daily_uniques"
	MonthlyUniqueCidsPath = "/monthly_uniques"

	ContentTypeForm = "application/x-www-form-urlencoded"

	ConfigFileFlag = "config"
	ConfigFilePath = "./config.yml"

	DbDriverName = "mysql"
)

// Config struct for app configuration file
type Config struct {
	Mysql struct {
		Host           string `yaml:"host"`
		Port           string `yaml:"port"`
		Username       string `yaml:"username"`
		Password       string `yaml:"password"`
		Name           string `yaml:"name"`
		MaxConnections int    `yaml:"max_connections"`
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
	db, err := NewDbStorageService(config)
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

func newRouter(db StorageService) *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc(CollectCidsPath, collectCid(db)).Methods(http.MethodGet)
	r.HandleFunc(DailyUniqueCidsPath, dailyUniqueCidCount(db)).Methods(http.MethodGet)
	r.HandleFunc(MonthlyUniqueCidsPath, monthlyUniqueCidCount(db)).Methods(http.MethodGet)

	return r
}

// collectCid registers a single client ID
func collectCid(db StorageService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", ContentTypeForm)
		query := r.URL.Query()

		// Parse client ID
		cid, err := uuid.Parse(query.Get(CidParam))
		if err != nil {
			log.Printf("Bad client ID %q: %v", cid, err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Parse date
		tm := time.Now().UTC()
		date := query.Get(DateParam)
		if date != "" {
			i, err := strconv.ParseInt(date, 10, 64)
			if err != nil {
				log.Printf("Bad date %q: %v", date, err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			tm = time.Unix(i, 0)
		}

		// Save client ID to the storage
		if err = db.SetCid(cid, tm); err != nil {
			log.Printf("Failed to save client ID and date %q %q: %v", cid, date, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func dailyUniqueCidCount(db StorageService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", ContentTypeForm)
		query := r.URL.Query()

		// Parse date
		date := query.Get(DateParam)
		tm, err := time.Parse("20200909", date)
		if err != nil {
			log.Printf("Bad date %q: %v", date, err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Get daily unique client IDs from the storage
		count, err := db.GetUniqueDailyCidCount(tm)
		if err != nil {
			log.Printf("Failed to get unique daily client ID count for date %q: %v", date, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if _, err = w.Write([]byte(fmt.Sprintf("%d", count))); err != nil {
			log.Printf(
				"Failed to write a response with daily unique client ID count %q: %v",
				count, err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func monthlyUniqueCidCount(db StorageService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", ContentTypeForm)
		query := r.URL.Query()

		// Parse date
		date := query.Get(DateParam)
		tm, err := time.Parse("20200909", date)
		if err != nil {
			log.Printf("Bad date %q: %v", date, err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Get monthly unique client IDs from the storage
		count, err := db.GetUniqueMonthlyCidCount(tm)
		if err != nil {
			log.Printf("Failed to get unique monthly client ID count for date %q: %v", date, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if _, err = w.Write([]byte(fmt.Sprintf("%d", count))); err != nil {
			log.Printf(
				"Failed to write a response with monthly unique client ID count %q: %v",
				count, err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}
