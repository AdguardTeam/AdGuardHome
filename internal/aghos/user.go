package aghos

// SetGroup sets the effective group ID of the calling process.
func SetGroup(groupName string) (err error) {
	return setGroup(groupName)
}

// SetUser sets the effective user ID of the calling process.
func SetUser(userName string) (err error) {
	return setUser(userName)
}
