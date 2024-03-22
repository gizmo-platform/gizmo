package ds

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

const (
	hostAPdConf = "/etc/hostapd/hostapd.conf"
	dnsmasqConf = "/etc/dnsmasq/dnsmasq.conf"
)

// Configure installs configuration files into the correct locations
// to permit operation of the network components.  It also restarts
// services as necessary.
func (ds *DriverStation) Configure() error {
	steps := []ConfigureStep{
		ds.configureHostname,
		ds.configureHostAPd,
		ds.configureDNSMasq,
		ds.enableServices,
	}
	names := []string{"hostname", "hostapd", "dnsmasq", "enable"}

	for i, step := range steps {
		ds.l.Info("Configuring", "step", names[i])
		if err := step(); err != nil {
			return err
		}
	}

	return nil
}

func (ds *DriverStation) configureHostname() error {
	f, err := os.Create("/etc/hostname")
	if err != nil {
		return err
	}
	fmt.Fprintf(f, "gizmoDS-%d\n", ds.cfg.Team)
	f.Close()

	if err := exec.Command("hostname", fmt.Sprintf("%d", ds.cfg.Team)).Run(); err != nil {
		return err
	}

	return nil
}

func (ds *DriverStation) configureHostAPd() error {
	if err := ds.doTemplate(hostAPdConf, "tpl/hostapd.conf.tpl", ds.cfg); err != nil {
		return err
	}
	return nil
}

func (ds *DriverStation) configureDNSMasq() error {
	if err := ds.doTemplate(dnsmasqConf, "tpl/dnsmasq.conf.tpl", ds.cfg); err != nil {
		return err
	}
	return nil
}

func (ds *DriverStation) enableServices() error {
	ds.svc.Enable("dnsmasq")
	ds.svc.Enable("hostapd")
	return nil
}

func (ds *DriverStation) doTemplate(path, source string, data interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		ds.l.Error("Error creating target template path", "path", path, "error", err)
		return err
	}

	fMap := template.FuncMap{
		"ip4prefix": ip4prefix,
	}

	tmpl, err := template.New(filepath.Base(source)).Funcs(fMap).ParseFS(efs, source)
	if err != nil {
		ds.l.Error("Error parsing template", "source", source, "error", err)
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		ds.l.Error("Error creating target file", "file", path, "error", err)
		return err
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		ds.l.Error("Error executing template", "data", data, "error", err)
		return err
	}

	return nil
}

func ip4prefix(t int) string {
	return fmt.Sprintf("10.%d.%d", int(t/100), t%100)
}
