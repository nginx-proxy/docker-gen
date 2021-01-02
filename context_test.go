package dockergen

import (
	"testing"
)

func TestGetCurrentContainerID(t *testing.T) {
	currentContainerID := GetCurrentContainerID()

	if len(currentContainerID) != 0 && len(currentContainerID) != 64 {
		t.Fail()
	}
}

func TestGetCurrentContainerID_ECS(t *testing.T) {
	cgroup :=
		`9:perf_event:/ecs/628967a1-46b4-4a8a-84ff-605128f4679e/3c94e08259a6235781bb65f3dec91150c92e9d414ecc410d6245687392d3900f
8:memory:/ecs/628967a1-46b4-4a8a-84ff-605128f4679e/3c94e08259a6235781bb65f3dec91150c92e9d414ecc410d6245687392d3900f
7:hugetlb:/ecs/628967a1-46b4-4a8a-84ff-605128f4679e/3c94e08259a6235781bb65f3dec91150c92e9d414ecc410d6245687392d3900f
6:freezer:/ecs/628967a1-46b4-4a8a-84ff-605128f4679e/3c94e08259a6235781bb65f3dec91150c92e9d414ecc410d6245687392d3900f
5:devices:/ecs/628967a1-46b4-4a8a-84ff-605128f4679e/3c94e08259a6235781bb65f3dec91150c92e9d414ecc410d6245687392d3900f
4:cpuset:/ecs/628967a1-46b4-4a8a-84ff-605128f4679e/3c94e08259a6235781bb65f3dec91150c92e9d414ecc410d6245687392d3900f
3:cpuacct:/ecs/628967a1-46b4-4a8a-84ff-605128f4679e/3c94e08259a6235781bb65f3dec91150c92e9d414ecc410d6245687392d3900f
2:cpu:/ecs/628967a1-46b4-4a8a-84ff-605128f4679e/3c94e08259a6235781bb65f3dec91150c92e9d414ecc410d6245687392d3900f
1:blkio:/ecs/628967a1-46b4-4a8a-84ff-605128f4679e/3c94e08259a6235781bb65f3dec91150c92e9d414ecc410d6245687392d3900f`

	if got, exp := matchECSCurrentContainerID(cgroup), "3c94e08259a6235781bb65f3dec91150c92e9d414ecc410d6245687392d3900f"; got != exp {
		t.Fatalf("id mismatch: got %v, exp %v", got, exp)
	}
}

func TestGetCurrentContainerID_DockerCE(t *testing.T) {
	cgroup :=
		`550 209 0:38 / / rw,relatime master:97 - overlay overlay rw,seclabel,lowerdir=/var/lib/docker/overlay2/l/CMLIRGTLVPWATOT2BBGMMWTKPL:/var/lib/docker/overlay2/l/JBJ54L4BQPWTRHZVLNTFAIKIMU:/var/lib/docker/overlay2/l/GKWSGW3V5DTMUG5OV2QF2NDXZM:/var/lib/docker/overlay2/l/KKB7IBA5SXQ6GOHKPC3X2TVPDL,upperdir=/var/lib/docker/overlay2/d3c3d349e227bba0d9f66787a03a5ba8928e6fb8198e787d4d038425afc89fd0/diff,workdir=/var/lib/docker/overlay2/d3c3d349e227bba0d9f66787a03a5ba8928e6fb8198e787d4d038425afc89fd0/work
551 550 0:47 / /proc rw,nosuid,nodev,noexec,relatime - proc proc rw
552 550 0:48 / /dev rw,nosuid - tmpfs tmpfs rw,seclabel,size=65536k,mode=755,inode64
553 552 0:49 / /dev/pts rw,nosuid,noexec,relatime - devpts devpts rw,seclabel,gid=5,mode=620,ptmxmode=666
554 550 0:50 / /sys ro,nosuid,nodev,noexec,relatime - sysfs sysfs ro,seclabel
555 554 0:27 / /sys/fs/cgroup ro,nosuid,nodev,noexec,relatime - cgroup2 cgroup rw,seclabel,nsdelegate
556 552 0:46 / /dev/mqueue rw,nosuid,nodev,noexec,relatime - mqueue mqueue rw,seclabel
557 552 0:51 / /dev/shm rw,nosuid,nodev,noexec,relatime - tmpfs shm rw,seclabel,size=65536k,inode64
558 550 253:1 /var/lib/docker/containers/bc4ed21e220deb8e759970f0853854b868f0157d1b45caab844c5641435553a4/resolv.conf /etc/resolv.conf rw,relatime - xfs /dev/mapper/fedora_tpdachsserver-root rw,seclabel,attr2,inode64,logbufs=8,logbsize=32k,noquota
559 550 253:1 /var/lib/docker/containers/bc4ed21e220deb8e759970f0853854b868f0157d1b45caab844c5641435553a4/hostname /etc/hostname rw,relatime - xfs /dev/mapper/fedora_tpdachsserver-root rw,seclabel,attr2,inode64,logbufs=8,logbsize=32k,noquota
560 550 253:1 /var/lib/docker/containers/bc4ed21e220deb8e759970f0853854b868f0157d1b45caab844c5641435553a4/hosts /etc/hosts rw,relatime - xfs /dev/mapper/fedora_tpdachsserver-root rw,seclabel,attr2,inode64,logbufs=8,logbsize=32k,noquota
210 551 0:47 /bus /proc/bus ro,relatime - proc proc rw
211 551 0:47 /fs /proc/fs ro,relatime - proc proc rw
212 551 0:47 /irq /proc/irq ro,relatime - proc proc rw
264 551 0:47 /sys /proc/sys ro,relatime - proc proc rw
265 551 0:47 /sysrq-trigger /proc/sysrq-trigger ro,relatime - proc proc rw
523 551 0:58 / /proc/asound ro,relatime - tmpfs tmpfs ro,seclabel,inode64
524 551 0:59 / /proc/acpi ro,relatime - tmpfs tmpfs ro,seclabel,inode64
529 551 0:48 /null /proc/kcore rw,nosuid - tmpfs tmpfs rw,seclabel,size=65536k,mode=755,inode64
530 551 0:48 /null /proc/keys rw,nosuid - tmpfs tmpfs rw,seclabel,size=65536k,mode=755,inode64
531 551 0:48 /null /proc/latency_stats rw,nosuid - tmpfs tmpfs rw,seclabel,size=65536k,mode=755,inode64
532 551 0:48 /null /proc/timer_list rw,nosuid - tmpfs tmpfs rw,seclabel,size=65536k,mode=755,inode64
533 551 0:48 /null /proc/sched_debug rw,nosuid - tmpfs tmpfs rw,seclabel,size=65536k,mode=755,inode64
534 551 0:60 / /proc/scsi ro,relatime - tmpfs tmpfs ro,seclabel,inode64
535 554 0:61 / /sys/firmware ro,relatime - tmpfs tmpfs ro,seclabel,inode64`

	if got, exp := matchDockerCurrentContainerID(cgroup), "bc4ed21e220deb8e759970f0853854b868f0157d1b45caab844c5641435553a4"; got != exp {
		t.Fatalf("id mismatch: got %v, exp %v", got, exp)
	}

}
