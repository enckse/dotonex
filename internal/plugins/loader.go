package plugins

import (
	"fmt"

	"voidedtech.com/radiucal/internal"
	"voidedtech.com/radiucal/internal/plugins/access"
	"voidedtech.com/radiucal/internal/plugins/log"
	"voidedtech.com/radiucal/internal/plugins/usermac"
)

// LoadPlugin loads a plugin from the name and into a module object
func LoadPlugin(name string, ctx *internal.PluginContext) (internal.Module, error) {
	mod, err := getPlugin(name)
	if err != nil {
		return nil, err
	}
	if err := mod.Setup(ctx.CloneContext()); err != nil {
		return nil, err
	}
	return mod, nil
}

func getPlugin(name string) (internal.Module, error) {
	switch name {
	case "usermac":
		return &usermac.Plugin, nil
	case "log":
		return &log.Plugin, nil
	case "access":
		return &access.Plugin, nil
	}
	return nil, fmt.Errorf("unknown plugin type %s", name)
}
