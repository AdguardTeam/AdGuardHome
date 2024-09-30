package aghos

// ConfigureSyslog reroutes standard logger output to syslog.
func ConfigureSyslog(serviceName string) (err error) {
	return configureSyslog(serviceName)
}
