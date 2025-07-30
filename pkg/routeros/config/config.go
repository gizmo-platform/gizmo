package config

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/vishvananda/netlink"

	"github.com/gizmo-platform/gizmo/pkg/eventstream"
)

const (
	workspaceFile = "main.tf"

	// BootstrapAddr points to where the scoring router would be
	// during the bootstrap scenario.
	BootstrapAddr = "100.64.1.1"

	// NormalAddr points to where the scoring router is during
	// normal operation.
	NormalAddr = "100.64.0.1"
)

// New initializes and returns a configurator
func New(opts ...Option) *Configurator {
	c := new(Configurator)
	c.stateDir = ".netstate"
	c.routerAddr = NormalAddr
	c.ctx = make(map[string]interface{})
	c.es = eventstream.NewNullStreamer()
	c.cl = &http.Client{
		Timeout: time.Second * 10,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	for _, o := range opts {
		o(c)
	}
	return c
}

// Zap destructively removes all state information that is used by the
// controller.
func (c *Configurator) Zap() error {
	return os.RemoveAll(c.stateDir)
}

// RadioMode returns the configured radio mode.  This is useful for
// passing to other systems to determine what mode they should be in
// based on what mode the FMS radio is in.
func (c *Configurator) RadioMode() string {
	return c.fc.RadioMode
}

// RadioChannelForField returns the currently configured radio channel
// for a given field.
func (c *Configurator) RadioChannelForField(id int) string {
	for _, f := range c.fc.Fields {
		if f.ID == id {
			return f.Channel
		}
	}
	return "AUTO"
}

// SyncState pushes the in-memory state down to the disk.
func (c *Configurator) SyncState(ctx map[string]interface{}) error {
	if err := os.MkdirAll(c.stateDir, 0755); err != nil {
		c.l.Warn("Couldn't make state directory", "error", err)
		return err
	}

	if err := c.extractModules(); err != nil {
		c.l.Warn("Couldn't extract modules", "error", err)
		return err
	}

	if err := c.configureWorkspace(ctx); err != nil {
		c.l.Warn("Couldn't configure workspace", "error", err)
		return err
	}

	if err := c.syncFMSConfig(); err != nil {
		c.l.Warn("Couldn't synchronize FMS config", "error", err)
		return err
	}

	return nil
}

// SyncTLM takes a mapping from the TLM and puts it down on disk so
// that a later converge run may act upon it.
func (c *Configurator) SyncTLM(tlm map[int]string) error {
	for team := range tlm {
		if _, ok := c.fc.Teams[team]; !ok {
			return fmt.Errorf("TLM requested an unknown team: %d", team)
		}
	}

	// This is a map of field number to map of port name to team VLAN ID
	fMap := make(map[int]map[string]int)

	for team, location := range tlm {
		parts := strings.Split(location, ":")
		fNum, _ := strconv.Atoi(strings.ReplaceAll(parts[0], "field", ""))
		if fMap[fNum] == nil {
			fMap[fNum] = make(map[string]int)
		}
		fMap[fNum][c.quadToEther(parts[1])] = c.fc.Teams[team].VLAN
	}

	f, err := os.Create(filepath.Join(c.stateDir, "tlm.json"))
	if err != nil {
		return err
	}
	defer f.Close()
	f.Chmod(0644)

	if err := json.NewEncoder(f).Encode(fMap); err != nil {
		return err
	}

	return nil
}

// Init performs initialization of the underlying terraform workspace
// to fetch plugins and initialize module links.
func (c *Configurator) Init() error {
	cmd := exec.Command("terraform", "init", "-no-color")
	cmd.Dir = c.stateDir
	rPipe, wPipe := io.Pipe()
	cmd.Stdout = wPipe
	cmd.Stderr = wPipe

	c.es.PublishLogLine("[LOG] => Initializing Workspace")
	cmd.Start()

	scanner := bufio.NewScanner(rPipe)
	scanner.Split(bufio.ScanLines)
	go func() {
		for scanner.Scan() {
			c.l.Info(scanner.Text())
		}
		c.l.Debug("Log copier shutting down")
	}()
	if err := cmd.Wait(); err != nil {
		return err
	}
	c.es.PublishLogLine("[LOG] => Workspace initialization complete")
	return nil
}

// Converge commands all network hardware to achieve the state
// currently on disk.
func (c *Configurator) Converge(refresh bool, target string) error {
	opts := []string{"apply", "-auto-approve", "-no-color"}
	if !refresh {
		opts = append(opts, "-refresh=false")
	}
	if target != "" {
		opts = append(opts, "-target="+target)
	}
	cmd := exec.Command("terraform", opts...)
	cmd.Dir = c.stateDir
	rPipe, wPipe := io.Pipe()
	cmd.Stdout = wPipe
	cmd.Stderr = wPipe

	cmd.Start()

	scanner := bufio.NewScanner(rPipe)
	scanner.Split(bufio.ScanLines)
	go func() {
		for scanner.Scan() {
			c.l.Info(scanner.Text())
		}
		c.l.Debug("Log copier shutting down")
	}()
	return cmd.Wait()
}

// ReprovisionCAP removes all CAP interfaces and then triggers a
// provisioning cycle.
func (c *Configurator) ReprovisionCAP() error {
	req := &http.Request{
		Method: http.MethodGet,
		URL: &url.URL{
			Scheme: "http",
			Host:   c.routerAddr,
			Path:   "/rest/caps-man/interface",
			User:   url.UserPassword(c.fc.AutoUser, c.fc.AutoPass),
		},
	}
	resp, err := c.cl.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	capList := []*rosCapInterface{}
	if err := json.NewDecoder(resp.Body).Decode(&capList); err != nil {
		return err
	}

	for _, capInterface := range capList {
		if capInterface.Master != "true" {
			continue
		}

		req = &http.Request{
			Method: http.MethodDelete,
			URL: &url.URL{
				Scheme:  "http",
				Host:    c.routerAddr,
				Path:    "/rest/caps-man/interface/" + capInterface.ID,
				RawPath: "/rest/caps-man/interface/" + capInterface.ID,
				User:    url.UserPassword(c.fc.AutoUser, c.fc.AutoPass),
			},
		}
		c.l.Debug("Proposed delete URL", "url", req.URL.String())
		resp, err := c.cl.Do(req)
		if err != nil {
			c.l.Error("Error removing cap interface", "cap", capInterface.ID, "error", err)
			return err
		}
		if resp.StatusCode != 204 {
			c.l.Error("CAP was not removed!", "code", resp.StatusCode)
		}
		defer resp.Body.Close()
	}

	req = &http.Request{
		Method: http.MethodGet,
		URL: &url.URL{
			Scheme: "http",
			Host:   c.routerAddr,
			Path:   "/rest/caps-man/remote-cap",
			User:   url.UserPassword(c.fc.AutoUser, c.fc.AutoPass),
		},
	}

	resp, err = c.cl.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	rCapList := []*rosRemoteCap{}
	if err := json.NewDecoder(resp.Body).Decode(&rCapList); err != nil {
		return err
	}

	for _, rCap := range rCapList {
		data, err := json.Marshal(rCap)
		if err != nil {
			return err
		}

		req.URL.Path = "/rest/caps-man/remote-cap/provision"
		req, _ := http.NewRequest("POST", req.URL.String(), bytes.NewBuffer(data))
		req.Header.Set("Content-Type", "application/json")
		resp, err := c.cl.Do(req)
		if err != nil {
			c.l.Error("Error triggering provisioning", "error", err)
			return err
		}
		defer resp.Body.Close()
	}

	return nil
}

// CycleRadio forces a provisioning cycle on the given band.
func (c *Configurator) CycleRadio(band string) error {
	// This only needs to happen if the field radio is the one
	// that is in use.  If its not, other mechanisms are in play.
	if c.fc.RadioMode != "FIELD" {
		return nil
	}

	req := &http.Request{
		Method: http.MethodGet,
		URL: &url.URL{
			Scheme: "http",
			Path:   "/rest/interface/wireless",
			User:   url.UserPassword(c.fc.AutoUser, c.fc.AutoPass),
		},
	}

	ifList := []*rosInterface{}
	for _, field := range c.fc.Fields {
		req.URL.Host = field.IP
		resp, err := c.cl.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		ifListTmp := []*rosInterface{}
		if err := json.NewDecoder(resp.Body).Decode(&ifListTmp); err != nil {
			return err
		}
		ifList = append(ifList, ifListTmp...)
	}
	c.l.Debug("Identified interfaces", "interfaces", ifList)

	req.URL.Host = c.routerAddr
	req.URL.Path = "/rest/caps-man/radio"
	resp, err := c.cl.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	capList := []*rosCapInterface{}
	if err := json.NewDecoder(resp.Body).Decode(&capList); err != nil {
		return err
	}

	for _, rosInterface := range ifList {
		if strings.HasPrefix(rosInterface.Band, band) {
			for _, capInterface := range capList {
				if capInterface.MAC == rosInterface.MAC {
					c.l.Debug("Found matching CAP", "id", capInterface.ID, "mac", capInterface.MAC)
					capInterface.MAC = "" // Zero out so it doesn't get sent back
					capInterface.Master = ""
					data, err := json.Marshal(capInterface)
					if err != nil {
						return err
					}
					c.l.Debug("JSON Message", "msg", string(data))

					req.URL.Path = "/rest/caps-man/radio/provision"
					req, _ := http.NewRequest("POST", req.URL.String(), bytes.NewBuffer(data))
					req.Header.Set("Content-Type", "application/json")
					resp, err = c.cl.Do(req)
					if err != nil {
						c.l.Error("Error cycling radio", "radio", capInterface.MAC, "band", band, "error", err)
						return err
					}
					defer resp.Body.Close()

					msg, _ := io.ReadAll(resp.Body)
					c.l.Debug("Cycle response", "resp", string(msg))
				}
			}
		}
	}

	return nil
}

// ActivateBootstrapNet raises VLAN2 with special addressing and
// allows the FMS processes to talk to the network elements prior to
// bootstrapping being complete.
func (c *Configurator) ActivateBootstrapNet() error {
	fmsAddr := "100.64.1.2"
	c.l.Info("Bootstrap mode enabled")

	eth0, err := netlink.LinkByName("eth0")
	if err != nil {
		c.l.Error("Could not retrieve ethernet link", "error", err)
		return err
	}

	bootstrap0 := &netlink.Vlan{
		LinkAttrs:    netlink.LinkAttrs{Name: "bootstrap0", ParentIndex: eth0.Attrs().Index},
		VlanId:       2,
		VlanProtocol: netlink.VLAN_PROTOCOL_8021Q,
	}

	if err := netlink.LinkAdd(bootstrap0); err != nil && err.Error() != "file exists" {
		c.l.Error("Could not create bootstrapping interface", "error", err)
		return err
	}

	for _, int := range []netlink.Link{eth0, bootstrap0} {
		if err := netlink.LinkSetUp(int); err != nil {
			c.l.Error("Error enabling eth0", "error", err)
			return err
		}
	}

	addr, _ := netlink.ParseAddr(fmsAddr + "/24")
	if err := netlink.AddrAdd(bootstrap0, addr); err != nil {
		c.l.Error("Could not add IP", "error", err)
		return err
	}
	return nil
}

// DeactivateBootstrapNet undoes what is setup by ActivateBootstrapNet
// and returns the system to its normal network operations.
func (c *Configurator) DeactivateBootstrapNet() error {
	bootstrap0, err := netlink.LinkByName("bootstrap0")
	if err != nil {
		c.l.Error("Could not retrieve ethernet link", "error", err)
		return err
	}

	if err := netlink.LinkDel(bootstrap0); err != nil {
		c.l.Error("Error removing bootstrap link", "error", err)
		return err
	}
	return nil
}

// ROSPing reaches out to a device and attempts to retrive its
// identity.  This validates that the device is up to a point that ROS
// can respond to API calls.
func (c *Configurator) ROSPing(addr, user, pass string) error {
	req := &http.Request{
		Method: http.MethodGet,
		URL: &url.URL{
			Scheme: "http",
			Host:   addr,
			Path:   "/rest/system/identity",
			User:   url.UserPassword(user, pass),
		},
	}
	_, err := c.cl.Do(req)
	return err
}

// GetIDOnPort checks for an LLDP identity on the given port and then
// resolves the number assuming that it is of the form 'gizmoDS-NNNN'.
func (c *Configurator) GetIDOnPort(field int, quad string) (int, error) {
	fIP := ""
	for _, f := range c.fc.Fields {
		if f.ID == field {
			fIP = f.IP
		}
	}
	if fIP == "" {
		c.l.Error("Bad field for IDonPort", "field", field, "quad", quad)
		return -1, errors.New("bad field spec")
	}
	req := &http.Request{
		Method: http.MethodGet,
		URL: &url.URL{
			Scheme: "http",
			Host:   fIP,
			Path:   "/rest/ip/neighbor",
			User:   url.UserPassword(c.fc.AutoUser, c.fc.AutoPass),
		},
	}
	q := req.URL.Query()
	q.Add(".proplist", "interface,identity")
	req.URL.RawQuery = q.Encode()
	resp, err := c.cl.Do(req)
	if err != nil {
		return -1, err
	}
	res := []rosNeighbor{}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return -1, err
	}

	for _, neighbor := range res {
		if strings.Contains(neighbor.Interface, c.quadToEther(quad)) {
			num, err := strconv.Atoi(strings.TrimPrefix(neighbor.Identity, "gizmoDS-"))
			if err != nil {
				return -1, err
			}
			return num, nil
		}
	}
	return -1, errors.New("not found")
}

func (c *Configurator) syncFMSConfig() error {
	f, err := os.Create(filepath.Join(c.stateDir, "fms.json"))
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(c.fc)
}

func (c *Configurator) extractModules() error {
	sub, _ := fs.Sub(efs, "tf/mod")
	return fs.WalkDir(sub, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			c.l.Debug("Creating Directory", "dir", path)
			if err := os.MkdirAll(filepath.Join(c.stateDir, "mod", path), 0755); err != nil {
				return err
			}
		} else {
			c.l.Debug("Extracting File", "file", path)
			src, err := sub.Open(path)
			if err != nil {
				return err
			}
			defer src.Close()

			dst, err := os.Create(filepath.Join(c.stateDir, "mod", path))
			if err != nil {
				return err
			}
			defer dst.Close()
			if _, err := io.Copy(dst, src); err != nil {
				return err
			}
		}
		return nil
	})
}

func (c *Configurator) convergeFields() error {
	for _, field := range c.fc.Fields {
		if err := c.waitForROS(field.IP, c.fc.AutoUser, c.fc.AutoPass); err != nil {
			return err
		}
		provisionFunc := func() error {
			if err := c.Converge(false, fmt.Sprintf("module.field%d", field.ID)); err != nil {
				c.l.Error("Error configuring field", "field", field.ID, "error", err)
				return err
			}
			return nil
		}

		bo := backoff.NewExponentialBackOff(backoff.WithMaxElapsedTime(time.Minute * 5))
		if err := backoff.Retry(provisionFunc, bo); err != nil {
			c.l.Error("Permanent error while configuring field", "error", err)
			return err
		}
	}
	return nil
}

func (c *Configurator) waitForROS(addr, user, pass string) error {
	retryFunc := func() error {
		if err := c.ROSPing(addr, user, pass); err != nil {
			c.l.Info("Waiting for device", "address", addr, "error", err)
			return err
		}
		return nil
	}

	if err := backoff.Retry(retryFunc, backoff.NewExponentialBackOff(backoff.WithMaxInterval(time.Second*30))); err != nil {
		c.l.Error("Permanent error encountered while waiting for RouterOS", "error", err)
		c.l.Error("You need to reboot network boxes and try again")
		return err
	}
	return nil
}

func (c *Configurator) waitForFMSIP() error {
	c.l.Debug("Aquiring eth0")
	eth0, err := netlink.LinkByName("eth0")
	if err != nil {
		c.l.Error("Could not retrieve ethernet link", "error", err)
		return err
	}

	fmsIP, _ := netlink.ParseAddr("100.64.0.2/24")

	retryFunc := func() error {
		c.l.Debug("Requesting addresses from eth0")
		addrs, err := netlink.AddrList(eth0, netlink.FAMILY_V4)
		if err != nil {
			c.l.Error("Error listing addresses", "error", err)
			return err
		}

		for _, a := range addrs {
			c.l.Debug("Checking IP", "have", a.String(), "want", fmsIP.String())
			if a.Equal(*fmsIP) {
				return nil
			}
		}

		return errors.New("No FMS IP")
	}
	c.l.Debug("Link acquired, waiting for address")

	if err := backoff.Retry(retryFunc, backoff.NewExponentialBackOff(backoff.WithMaxInterval(time.Second*30))); err != nil {
		c.l.Error("Permanent error encountered while waiting for dhcp address", "error", err)
		c.l.Error("You may be able to recover from this by restarting dhcpcd")
	}
	return nil
}

func (c *Configurator) configureWorkspace(ctx map[string]interface{}) error {
	if ctx == nil {
		ctx = make(map[string]interface{})
	}
	if _, found := ctx["RouterBootstrap"]; !found {
		ctx["RouterBootstrap"] = false
	}

	if _, found := ctx["FieldBootstrap"]; !found {
		ctx["FieldBootstrap"] = false
	}

	ctx["FMS"] = c.fc
	ctx["RouterAddr"] = c.routerAddr

	tmpl, err := template.New(workspaceFile).ParseFS(efs, filepath.Join("tf", workspaceFile))
	if err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(c.stateDir, workspaceFile))
	if err != nil {
		return err
	}
	defer f.Close()

	if err := tmpl.Execute(f, ctx); err != nil {
		return err
	}

	return nil
}

func (c *Configurator) quadToEther(quad string) string {
	switch quad {
	case "red":
		return "ether2"
	case "blue":
		return "ether3"
	case "green":
		return "ether4"
	case "yellow":
		return "ether5"
	}
	return ""
}
