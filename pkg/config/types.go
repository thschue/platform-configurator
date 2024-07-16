package config

import (
	"github.com/thschue/platformer/pkg/gitea"
	"github.com/thschue/platformer/pkg/harbor"
)

type Config struct {
	Gitea  gitea.Config  `yaml:"gitea"`
	Harbor harbor.Config `yaml:"harbor"`
}
