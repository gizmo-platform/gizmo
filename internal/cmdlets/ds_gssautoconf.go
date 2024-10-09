//go:build linux

package cmdlets

import (
	"encoding/json"
	"math/rand"
	"os"
	"strconv"
	"strings"

	"github.com/diskfs/go-diskfs"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/vishvananda/netlink"

	"github.com/gizmo-platform/gizmo/pkg/config"
)

var (
	dsGSSAutoConfCmd = &cobra.Command{
		Use:   "gss-autoconf",
		Short: "gss-autoconf attempts to automagically configure the driver's station using the volume label",
		Long:  dsGSSAutoConfCmdLongDocs,
		Run:   dsGSSAutoConfCmdRun,
	}

	dsGSSAutoConfCmdLongDocs = `gss-autoconf attempts to read the volume label off the first partition of /dev/mccblk0 and work out what team number the driver's station belongs to.

To use this feature, name the FAT32 filesystem with a label of the form GIZMO<TEAM> where <TEAM> is the number of the team you wish to associate with this Gizmo/DS pair.`
)

func init() {
	dsCmd.AddCommand(dsGSSAutoConfCmd)
}

func dsGSSAutoConfCmdRun(c *cobra.Command, args []string) {
	bindInterface := "eth0"
	diskPath := "/dev/mmcblk0p1"

	os.Exit(func() int {
		initLogger("gss-autoconf")
		d, err := diskfs.Open(diskPath)
		if err != nil {
			appLogger.Error("Error opening disk", "error", err)
			return 1
		}
		fs, err := d.GetFilesystem(0) // assuming it is the whole disk, so partition = 0
		if err != nil {
			appLogger.Error("Error opening filesystem", "error", err)
			return 1
		}

		label := strings.TrimSpace(fs.Label())
		if !strings.HasPrefix(label, "GIZMO") {
			appLogger.Warn("Volume label does not start with 'GIZMO'", "label", label)
			return 2
		}
		num, err := strconv.Atoi(strings.TrimPrefix(label, "GIZMO"))
		if err != nil {
			appLogger.Error("Couldn't parse number from label")
			return 2
		}

		eth0, err := netlink.LinkByName(bindInterface)
		if err != nil {
			appLogger.Error("Could not retrieve ethernet link", "error", err)
			return 2
		}
		mac := eth0.Attrs().HardwareAddr

		// This part looks really dumb and looks like it has
		// security implications because it is and it does.
		// This is used in a non-critical context to generate
		// the network parameters on the fly on each boot.  If
		// you want truly random values here, use a
		// gsscfg.json file.
		var seed int64
		seed = int64(num)
		for _, b := range mac {
			seed = seed * int64(b)
		}
		r := rand.New(rand.NewSource(seed))
		uuid.SetRand(r)

		cfg := config.Config{
			Team:             num,
			UseDriverStation: true,
			UseExtNet:        false,
			ServerIP:         "gizmo-ds",
			NetSSID:          strings.ReplaceAll(uuid.New().String(), "-", ""),
			NetPSK:           strings.ReplaceAll(uuid.New().String(), "-", ""),
		}

		f, err := os.Create("/boot/gsscfg.json")
		if err != nil {
			appLogger.Error("Could not create /boot/gsscfg.json", "error", err)
			return 1
		}
		defer f.Close()

		if err := json.NewEncoder(f).Encode(cfg); err != nil {
			appLogger.Error("Could not write /boot/gsscfg.json", "error", err)
			return 1
		}

		appLogger.Info("Generated ephemeral gsscfg.json", "team", num)
		return 0
	}())
}
