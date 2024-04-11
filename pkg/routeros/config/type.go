// Package config maintains all of the machinery and embedded
// terraform data to configure field routers and access points
package config

import (
	"embed"

	"github.com/hashicorp/go-hclog"

	"github.com/gizmo-platform/gizmo/pkg/fms"
)

//go:embed tf/*
var efs embed.FS

// Option configures the Configurator
type Option func(*Configurator)

// Configurator is a mechansim to drive terraform under the hood and
// validate that the configuration is as intended.
type Configurator struct {
	l  hclog.Logger
	fc fms.Config

	stateDir string

	routerAddr string
}
