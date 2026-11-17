//go:build windows

package manager

import (
	"path/filepath"

	"golang.org/x/sys/windows"
)

func diskSpace(path string) (total, free uint64, id string, err error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, 0, "", err
	}
	var avail, all, totalFree uint64
	if err = windows.GetDiskFreeSpaceEx(p, &avail, &all, &totalFree); err != nil {
		return 0, 0, "", err
	}
	return all, avail, filepath.VolumeName(path), nil
}
