package build

// Commands is a definition of commands available in build system
var Commands = map[string]interface{}{
	"tools/build":   buildMe,
	"git/fetch":     gitFetch,
	"dev/goimports": goImports,
	"dev/lint":      goLint,
	"dev/test":      goTest,
	"build":         buildApp,
	"run":           runApp,
}
