package vhostzfs

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	zfs "github.com/bicomsystems/go-libzfs"
)



func SetupDiskZfs(ctx context.Context, target string) error {
	/* TODO: use go-libzfs package to create pool */
	cmd := exec.Command("zpool",
		"create",
		"tank",
		"-f",
		target)

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed creating zfs pool: %v", err)
	}

	pool, err := zfs.PoolOpen("tank")
	if err != nil {
		return fmt.Errorf("failed creating zfs pool: %v", err)
	}
	pool.Close()

	props := make(map[zfs.Prop]zfs.Property)
	strSize := fmt.Sprintf("%d", 1024*1024*1024*60)
	props[zfs.DatasetPropVolsize] = zfs.Property{Value: strSize}
	props[zfs.DatasetPropVolblocksize] = zfs.Property{Value: fmt.Sprintf("%d", 16*1024)}
	props[zfs.DatasetPropReservation] = zfs.Property{Value: strSize}

	dataset, err := zfs.DatasetCreate("tank/test-zvol", zfs.DatasetTypeVolume, props)
	if err != nil {
		return fmt.Errorf("Failed to create zvol: %w", err)
	}
	defer dataset.Close()

	return nil
}

func waitForFile(fileName string) error {
	maxDelay := time.Second * 5
	delay := time.Millisecond * 500
	var waited time.Duration
	for {
		if delay != 0 {
			time.Sleep(delay)
			waited += delay
		}
		if _, err := os.Stat(fileName); err == nil {
			return nil
		} else {
			if waited > maxDelay {
				return fmt.Errorf("file not found: error %v", err)
			}
			delay = 2 * delay
			if delay > maxDelay {
				delay = maxDelay
			}
		}
	}
}

const (
	tgtPath    = "/sys/kernel/config/target"
	iBlockPath = tgtPath + "/core/iblock_0"
	naaPrefix  = "5001405" // from rtslib-fb
)

// VHostCreateIBlock - Create vHost fabric
func VHostCreateIBlock(tgtName, wwn string) error {
	targetRoot := filepath.Join(iBlockPath, tgtName)
	if _, err := os.Stat(targetRoot); err != nil {
		return fmt.Errorf("tgt access error (%s): %s", targetRoot, err)
	}
	vhostRoot := filepath.Join(tgtPath, "vhost", wwn, "tpgt_1")
	vhostLun := filepath.Join(vhostRoot, "lun", "lun_0")
	err := os.MkdirAll(vhostLun, os.ModeDir)
	if err != nil {
		return fmt.Errorf("cannot create vhost: %v", err)
	}
	controlCommand := "scsi_host_id=1,scsi_channel_id=0,scsi_target_id=0,scsi_lun_id=0"
	if err := ioutil.WriteFile(filepath.Join(targetRoot, "control"), []byte(controlCommand), 0660); err != nil {
		return fmt.Errorf("error set control: %v", err)
	}
	if err := waitForFile(filepath.Join(vhostRoot, "nexus")); err != nil {
		return fmt.Errorf("error waitForFile: %v", err)
	}
	if err := ioutil.WriteFile(filepath.Join(vhostRoot, "nexus"), []byte(wwn), 0660); err != nil {
		return fmt.Errorf("error set nexus: %v", err)
	}
	if _, err := os.Stat(filepath.Join(vhostLun, "iblock")); os.IsNotExist(err) {
		if err := os.Symlink(targetRoot, filepath.Join(vhostLun, "iblock")); err != nil {
			return fmt.Errorf("error create symlink: %v", err)
		}
	}
	return nil
}

func VHostDeleteIBlock(wwn string) error {
	vhostRoot := filepath.Join(tgtPath, "vhost", wwn, "tpgt_1")
	vhostLun := filepath.Join(vhostRoot, "lun", "lun_0")
	if _, err := os.Stat(vhostLun); os.IsNotExist(err) {
		return fmt.Errorf("vHost do not exists for wwn %s: %s", wwn, err)
	}
	if err := os.Remove(filepath.Join(vhostLun, "iblock")); err != nil {
		return fmt.Errorf("error delete symlink: %v", err)
	}
	if err := os.RemoveAll(vhostLun); err != nil {
		return fmt.Errorf("error delete lun: %v", err)
	}
	if err := os.RemoveAll(vhostRoot); err != nil {
		return fmt.Errorf("error delete lun: %v", err)
	}
	if err := os.RemoveAll(filepath.Dir(vhostRoot)); err != nil {
		return fmt.Errorf("error delete lun: %v", err)
	}
	return nil
}

func TargetCreateIBlock(dev, tgtName, serial string) error {
	targetRoot := filepath.Join(iBlockPath, tgtName)
	err := os.MkdirAll(targetRoot, os.ModeDir)
	if err != nil {
		return fmt.Errorf("cannot create fileio: %v", err)
	}
	if err := waitForFile(filepath.Join(targetRoot, "control")); err != nil {
		return fmt.Errorf("error waitForFile: %v", err)
	}
	controlCommand := fmt.Sprintf("udev_path=%s", dev)
	if err := ioutil.WriteFile(filepath.Join(targetRoot, "control"), []byte(controlCommand), 0660); err != nil {
		return fmt.Errorf("error set control: %v", err)
	}
	if err := ioutil.WriteFile(filepath.Join(targetRoot, "wwn", "vpd_unit_serial"), []byte(serial), 0660); err != nil {
		return fmt.Errorf("error set vpd_unit_serial: %v", err)
	}
	if err := ioutil.WriteFile(filepath.Join(targetRoot, "enable"), []byte("1"), 0660); err != nil {
		return fmt.Errorf("error set enable: %v", err)
	}
	return nil
}

func GetSerialTarget(tgtName string) (string, error) {
	targetRoot := filepath.Join(iBlockPath, tgtName)
	//it returns something like "T10 VPD Unit Serial Number: 5001405043a8fbf4"
	serial, err := ioutil.ReadFile(filepath.Join(targetRoot, "wwn", "vpd_unit_serial"))
	if err != nil {
		return "", fmt.Errorf("GetSerialTarget for %s: %s", targetRoot, err)
	}
	parts := strings.Fields(strings.TrimSpace(string(serial)))
	if len(parts) == 0 {
		return "", fmt.Errorf("GetSerialTarget for %s: empty line", targetRoot)
	}
	return parts[len(parts)-1], nil
}

func IsVhostIblockExist(tgtName string) (bool, error) {
	serial, err := GetSerialTarget(tgtName)
	if err != nil {
		return false, fmt.Errorf("CheckVHostIBlock (%s): %v", tgtName, err)
	}

	vhostRoot := filepath.Join(tgtPath, "vhost", fmt.Sprintf("naa.%s", serial), "tpgt_1")
	vhostLun := filepath.Join(vhostRoot, "lun", "lun_0")
	if _, err := os.Stat(filepath.Join(vhostLun, "iblock")); err == nil {
		return true, nil
	}
	return false, nil
}

func GenerateNaaSerial() string {
	return fmt.Sprintf("%s%09x", naaPrefix, rand.Uint32())
}

func SetupVhost(device, iblockId string) (string, error) {
	device = "/dev/zvol/tank/test-zvol"
	iblockId = "test_iblock"
	serial := GenerateNaaSerial()
	wwn := fmt.Sprintf("naa.%s", serial)
	err := TargetCreateIBlock(device, iblockId, serial)
	if err != nil {
		return "", fmt.Errorf("TargetCreateFileIODev(%s, %s, %s): %v",
			device, iblockId, serial, err)
	}
	exists,err := IsVhostIblockExist(iblockId)
	if !exists {
		err = VHostCreateIBlock(iblockId, wwn)
		if err != nil {
			errString := fmt.Sprintf("VHostCreateIBlock: %v", err)
			err = VHostDeleteIBlock(wwn)
			if err != nil {
				errString = fmt.Sprintf("%s; VHostDeleteIBlock: %v",
					errString, err)
			}
			return "", fmt.Errorf("VHostCreateIBlock(%s, %s): %s",
				iblockId, wwn, errString)
		}
	}
	return wwn, nil
}
