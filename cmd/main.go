package main

import (
	"github.com/wojciech-malota-wojcik/netdata"
	"github.com/wojciech-malota-wojcik/netdata/lib/run"
)

func main() {
	run.Service("digest", netdata.IoCBuilder, netdata.App)
}
