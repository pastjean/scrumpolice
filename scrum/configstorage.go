package scrum

type ConfigurationStorage interface {
	Load() *Config
	Save(config *Config)
}
