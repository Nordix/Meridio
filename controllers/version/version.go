package version

import "fmt"

var (
	Branch string
	Hash   string
)

// PrintBuildInfo prints the build information.
func VersionInfo() string {
	hasHashInfo := Hash != ""
	hasBranchInfo := Branch != ""
	switch {
	case hasBranchInfo && !hasHashInfo:
		return fmt.Sprintf("branch: %s", Branch)
	case !hasBranchInfo && hasHashInfo:
		return fmt.Sprintf("hash: %s", Hash)
	case hasBranchInfo && hasHashInfo:
		return fmt.Sprintf("branch: %s, hash: %s", Branch, Hash)
	default:
		return "no version info"
	}
}
