// Package fms encapsulates all the various parts of the Field
// Managent System, its associated configuration logic, and the APIs
// that talk to other systems.
package fms

import (
	nhttp "net/http"
	"sync"
	"time"

	"github.com/flosch/pongo2/v6"
	"github.com/hashicorp/go-hclog"

	"github.com/gizmo-platform/gizmo/pkg/config"
	"github.com/gizmo-platform/gizmo/pkg/http"
	"github.com/gizmo-platform/gizmo/pkg/routeros/netinstall"
)

// TeamLocationMapper looks at all teams trying to fetch a value and
// tries to get them controller based on their current match and their
// number.
type TeamLocationMapper interface {
	GetFieldForTeam(int) (string, error)
	GetCurrentMapping() (map[int]string, error)
	InsertOnDemandMap(map[int]string) error

	GetStageMapping() (map[int]string, error)
	InsertStageMapping(map[int]string) error
	CommitStagedMap() error
}

// EventStreamer represents the streaming interface server that allows
// websocket subscribers to join the event stream and get broadcast
// events.
type EventStreamer interface {
	Handler(nhttp.ResponseWriter, *nhttp.Request)

	PublishActionStart(string, string)
	PublishActionComplete(string)
	PublishError(error)
	PublishFileFetch(string)
	PublishLogLine(string)
}

// FileFetcher fetches restricted files that cannot be baked into the
// image.
type FileFetcher interface {
	FetchPackages() error
	FetchTools() error
}

type hudVersions struct {
	HardwareVersions string
	FirmwareVersions string
	Bootmodes        string
	DSVersions       string
}

// FMS encapsulates the FMS runnable.
type FMS struct {
	s  *http.Server
	c  *config.FMSConfig
	l  hclog.Logger
	es EventStreamer

	fetcher FileFetcher

	tlm TeamLocationMapper

	swg *sync.WaitGroup
	tpl *pongo2.TemplateSet

	quads []string

	stop           chan struct{}
	connectedDS    map[int]time.Time
	connectedGizmo map[int]time.Time
	connectedMutex *sync.RWMutex
	gizmoMeta      map[int]config.GizmoMeta
	dsMeta         map[int]config.DSMeta
	metaMutex      *sync.RWMutex

	netinst *netinstall.Installer

	hudVersions hudVersions
}
