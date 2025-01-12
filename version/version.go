package version

var (
	VersionSuffix = "DEV" // eg. DEV
	VersionTag    = "v0.1.4"
	Version       string
)

func init() {
	if Version == "" {
		if VersionSuffix == "" {
			Version = VersionTag
		} else {
			Version = VersionTag + "-" + VersionSuffix
		}
	}
}
