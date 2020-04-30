package version

import "fmt"

const logo = `

 ____  ____   ___   _____   ___    __  __ __  ______   ___   ____
|    \|    \ /   \ / ___/  /  _]  /  ]|  |  ||      | /   \ |    \
|  o  )  D  )     (   \_  /  [_  /  / |  |  ||      ||     ||  D  )
|   _/|    /|  O  |\__  ||    _]/  /  |  |  ||_|  |_||  O  ||    /
|  |  |    \|     |/  \ ||   [_/   \_ |  :  |  |  |  |     ||    \
|  |  |  .  \     |\    ||     \     ||     |  |  |  |     ||  .  \
|__|  |__|\_|\___/  \___||_____|\____| \__,_|  |__|   \___/ |__|\_|

`

const mark = `+----------------------+------------------------------------------+`

// These variables are populated via the Go linker.
var (
	UTCBuildTime  = "unknown"
	ClientVersion = "unknown"
	GoVersion     = "unknown"
	GitBranch     = "unknown"
	GitTag        = "unknown"
	GitHash       = "unknown"
)

var Version = fmt.Sprintf("%s\n%s\n| % -20s | % -40s |\n| % -20s | % -40s |\n| % -20s | % -40s |\n| % -20s | % -40s |\n| % -20s | % -40s |\n| % -20s | % -40s |\n%s\n",
	logo,
	mark,
	"Client Version", ClientVersion,
	"Go Version", GoVersion,
	"UTC Build Time", UTCBuildTime,
	"Git Branch", GitBranch,
	"Git Tag", GitTag,
	"Git Hash", GitHash,
	mark)
