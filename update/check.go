package update

type VersionInfo struct {
	NewVersion           string
	Announcement         string
	AnnouncementURL      string
	SelfUpdateMinVersion string
	CanAutoUpdate        bool
}

func (u *Updater) GetVersionResponse() (VersionInfo, error) {
	return VersionInfo{}, nil
}
