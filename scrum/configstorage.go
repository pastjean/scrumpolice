package scrum

type ConfigurationStorage interface {
	load() *Config
	save(config *Config)
}
