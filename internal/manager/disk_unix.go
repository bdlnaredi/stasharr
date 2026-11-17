//go:build !windows

package manager

import "syscall"

// diskSpace returns the size and free space of the disk holding path, and
// an id of the disk, so folders on the same disk are shown once.
func diskSpace(path string) (total, free uint64, id string, err error) {
	var st syscall.Statfs_t
	if err = syscall.Statfs(path, &st); err != nil {
		return 0, 0, "", err
	}
	bs := uint64(st.Bsize) //nolint:unconvert // Bsize is int32 on some systems
	var dev syscall.Stat_t
	if syscall.Stat(path, &dev) == nil {
		id = fmtUint(uint64(dev.Dev)) //nolint:unconvert // Dev is int32 on some systems
	}
	return st.Blocks * bs, st.Bavail * bs, id, nil
}
