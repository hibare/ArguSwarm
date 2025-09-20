package kubernetes

import (
	"github.com/hibare/ArguSwarm/internal/providers"
	"github.com/hibare/ArguSwarm/internal/providers/types"
)

func init() {
	providers.Registry.Register(types.ProviderKubernetes, NewProvider)
}
