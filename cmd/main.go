package main

import (
	"github.com/wojciech-malota-wojcik/ioc"
	digest "github.com/wojciech-malota-wojcik/netdata-digest"
	"github.com/wojciech-malota-wojcik/netdata-digest/lib/run"
)

func main() {
	run.Service("digest", digest.IoC, func(c *ioc.Container, configF *digest.ConfigFactory) error {
		return nil
	})
}
