package logs

type Api interface {
	Store
	Searcher
}

type Searcher interface {
	Search(*SearchParams) *LogsPayload
}

// Store
type Store interface {
	Start()

	// Close query log object
	Close()

	// Add a log entry
	Add(params *AddParams)

	// WriteDiskConfig - write configuration
	WriteDiskConfig(c *Config)

	// Get the config info that will be returned for config info
	ConfigInfo() *ConfigPayload

	// Clear the log
	Clear()

	// Config the logging implementation
	ApplyConfig(*ConfigPayload) error
}
