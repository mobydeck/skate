package version

// Version is set at build time via ldflags:
//
//	go build -ldflags "-X skate/internal/version.Version=1.0.0"
//
// If not set, defaults to "dev".
var Version = "dev"

func UserAgent() string {
	return "Skate/" + Version
}
