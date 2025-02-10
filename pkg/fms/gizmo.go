package fms

import (
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/gizmo-platform/gizmo/pkg/config"
	"github.com/gizmo-platform/gizmo/pkg/ds"
)

func (f *FMS) doConnectedUpkeep() {
	ticker := time.NewTicker(time.Second)

	for {
		select {
		case <-f.stop:
			ticker.Stop()
			return
		case <-ticker.C:
			f.connectedMutex.Lock()
			f.metaMutex.Lock()
			for id, expiry := range f.connectedDS {
				if time.Now().After(expiry) {

					delete(f.connectedDS, id)
					delete(f.dsMeta, id)
				}
			}
			for id, expiry := range f.connectedGizmo {
				if time.Now().After(expiry) {
					delete(f.connectedGizmo, id)
					delete(f.gizmoMeta, id)
				}
			}
			f.metaMutex.Unlock()
			f.connectedMutex.Unlock()
		}
	}
}

func (f *FMS) gizmoConfig(w http.ResponseWriter, r *http.Request) {
	tStr := chi.URLParam(r, "id")
	team, err := strconv.Atoi(tStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	location, err := f.tlm.GetFieldForTeam(team)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	parts := strings.SplitN(location, ":", 2)
	fnum, _ := strconv.Atoi(strings.ReplaceAll(parts[0], "field", ""))

	d := ds.FieldConfig{
		RadioMode:    f.c.RadioMode,
		RadioChannel: f.c.Fields[fnum-1].Channel,
		Field:        fnum,
		Location:     strings.ToUpper(parts[1]),
	}

	if err := json.NewEncoder(w).Encode(d); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (f *FMS) gizmoDSMetaReport(w http.ResponseWriter, r *http.Request) {
	tStr := chi.URLParam(r, "id")
	team, err := strconv.Atoi(tStr)
	if err != nil {
		f.l.Warn("Bad DS Meta report", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	d := config.DSMeta{}
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		f.l.Warn("Error deserializing DS Meta report", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	f.connectedMutex.Lock()
	f.connectedDS[team] = time.Now().Add(time.Second * 5)
	f.connectedMutex.Unlock()

	f.metaMutex.Lock()
	f.dsMeta[team] = d
	f.metaMutex.Unlock()
}

func (f *FMS) gizmoMetaReport(w http.ResponseWriter, r *http.Request) {
	tStr := chi.URLParam(r, "id")
	team, err := strconv.Atoi(tStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		f.l.Warn("Bad Gizmo Meta Report", "error", err)
		return
	}

	d := config.GizmoMeta{}
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		f.l.Warn("Error deserializing Gizmo Meta report", "error", err)
		return
	}

	f.connectedMutex.Lock()
	f.connectedGizmo[team] = time.Now().Add(time.Second * 5)
	f.connectedMutex.Unlock()

	f.metaMutex.Lock()
	f.gizmoMeta[team] = d
	f.metaMutex.Unlock()
}

func (f *FMS) gizmoUDPServelet() error {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.IPv4(100, 64, 0, 2),
		Port: 1729,
	})
	if err != nil {
		f.l.Error("Error binding UDP socket", "error", err)
		return err
	}

	go func() {
		<-f.stop
		conn.Close()
	}()
	buf := make([]byte, 1024)

	f.l.Info("UDP Listener Starting")
	for {
		n, a, err := conn.ReadFromUDP(buf)
		if err != nil {
			f.l.Warn("Error reading packet from UDP", "error", err)
			continue
		}

		team := int(a.IP[1])*100 + int(a.IP[2])

		switch rune(buf[0]) {
		case 'M':
			f.l.Trace("Gizmo Meta Buffer", "team", team, "buffer", string(buf))
			d := config.GizmoMeta{}
			if err := json.Unmarshal(buf[1:n], &d); err != nil {
				f.l.Warn("Error deserializing Gizmo Meta report", "error", err)
				continue
			}

			f.connectedMutex.Lock()
			f.connectedGizmo[team] = time.Now().Add(time.Second * 5)
			f.connectedMutex.Unlock()

			f.metaMutex.Lock()
			f.gizmoMeta[team] = d
			f.metaMutex.Unlock()
		}
	}
}
