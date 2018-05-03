package scrum

import (
	"os"
	"log"
	"encoding/json"
)

type FileConfigurationStorage struct {
	fileName *string
}

func (configStorage *FileConfigurationStorage) load() *Config {
	file, err := os.Open(*configStorage.fileName)
	if err != nil {
		log.Println("Cannot open file '", *configStorage.fileName, "', error:", err)
		return nil
	}

	var config Config

	err = json.NewDecoder(file).Decode(&config)
	if err != nil {
		log.Println("Cannot parse configuration file ('", *configStorage.fileName, "') content:", err)
	}
	return &config
	}

func (configStorage *FileConfigurationStorage) save(config *Config) {
	file, err := os.OpenFile(*configStorage.fileName, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		log.Println("Cannot open file '", *configStorage.fileName, "', error:", err)
		return
	}

	err = json.NewEncoder(file).Encode(&config)
	if err != nil {
		log.Println("Cannot serialize configuration file ('", *configStorage.fileName, "') content:", err)
	}
}

func NewFileConfigurationStorage(fileName *string) ConfigurationStorage {

	return &FileConfigurationStorage{fileName}
}