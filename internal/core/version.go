package core

import (
	_ "embed"
	"strings"
)

//go:generate sh -c "(git describe --tags --long --dirty='-devel' --match '[0-9]*.[0-9]*.[0-9]*' || echo '0.0.0') > version.txt"
//go:embed version.txt
var version string

var Version string = strings.TrimSpace(version)
