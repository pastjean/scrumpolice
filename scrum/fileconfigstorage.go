package scrum

import (
	"os"
	"log"
	"encoding/json"
	"bytes"
	"io"
)

type FileConfigurationStorage struct {
	fileName *string
}

func (configStorage *FileConfigurationStorage) Load() *Config {
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

func (configStorage *FileConfigurationStorage) Save(config *Config) {
	buffer := new(bytes.Buffer)
	err := json.NewEncoder(buffer).Encode(config)
	if err != nil {
		log.Println("Cannot serialize configuration state. content:", err)
		return
	}

	file, err := os.OpenFile(*configStorage.fileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		log.Println("Cannot open file '", *configStorage.fileName, "', error:", err)
		return
	}

	io.Writer(file).Write(buffer.Bytes())
}

func NewFileConfigurationStorage(fileName *string) ConfigurationStorage {

	return &FileConfigurationStorage{fileName}
}