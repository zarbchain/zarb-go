package version

import (
	"fmt"
)

// These constants follow the semantic versioning 2.0.0 spec (http://semver.org/)
const (
	major uint   = 1
	minor uint   = 0
	patch uint   = 1
	meta  string = ""
)

var build string

func Version() string {
	version := fmt.Sprintf("%d.%d.%d", major, minor, patch)

	if meta != "" {
		version = fmt.Sprintf("%s-%s", version, meta)
	}

	if build != "" {
		version = fmt.Sprintf("%s+%s", version, build)
	}

	return version
}
