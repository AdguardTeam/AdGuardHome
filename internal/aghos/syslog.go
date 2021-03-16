package aghos

// ConfigureSyslog reroutes standard logger output to syslog.
func ConfigureSyslog(serviceName string) error {
	return configureSyslog(serviceName)
}
