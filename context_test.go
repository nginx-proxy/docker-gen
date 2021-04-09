package dockergen

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"
)

func TestGetCurrentContainerID(t *testing.T) {
	hostname := os.Getenv("HOSTNAME")
	defer os.Setenv("HOSTNAME", hostname)

	ids := []string{
		"0fa939e22e6938e7517f663de83e79a5087a18b1b997a36e0c933a917cddb295",
		"e881f8c51a72db7da515e9d5cab8ed105b869579eb9923fdcf4ee80933160802",
		"eede6bd9e72f5d783a4bfb845bd71f310e974cb26987328a5d15704e23a8d6cb",
	}

	contents := map[string]string{
		"cpuset": fmt.Sprintf("/docker/%v", ids[0]),
		"cgroup": fmt.Sprintf(`13:name=systemd:/docker-ce/docker/%[1]v
12:pids:/docker-ce/docker/%[1]v
11:hugetlb:/docker-ce/docker/%[1]v
10:net_prio:/docker-ce/docker/%[1]v
9:perf_event:/docker-ce/docker/%[1]v
8:net_cls:/docker-ce/docker/%[1]v
7:freezer:/docker-ce/docker/%[1]v
6:devices:/docker-ce/docker/%[1]v
5:memory:/docker-ce/docker/%[1]v
4:blkio:/docker-ce/docker/%[1]v
3:cpuacct:/docker-ce/docker/%[1]v
2:cpu:/docker-ce/docker/%[1]v
1:cpuset:/docker-ce/docker/%[1]v`, ids[1]),
		"mountinfo": fmt.Sprintf(`705 661 0:96 / / rw,relatime master:192 - overlay overlay rw,lowerdir=/var/lib/docker/overlay2/l/CVAK3VWZFQCUGTLHRJHPEKJ4UL:/var/lib/docker/overlay2/l/XMJZ73SKVWVECU7TJCOY62F3H2:/var/lib/docker/overlay2/l/AVNBXO52GHDY3MZU3R4RCSNMCE:/var/lib/docker/overlay2/l/L4IJZ33E6NAMXJ5W3SKJSVX5TS:/var/lib/docker/overlay2/l/JXAUAD5TDJCXA34FGS6NYGUZKT:/var/lib/docker/overlay2/l/TBQDSAFKBSTFMUS3QCFWN5NRLB:/var/lib/docker/overlay2/l/MXIUXRGB7MU4Y4NUNZE2VXTXIN:/var/lib/docker/overlay2/l/HN7E4YWJG7TMG7BXLZTGICTBOA:/var/lib/docker/overlay2/l/65XQPC72Z5VRY4THGASZIQXS57:/var/lib/docker/overlay2/l/BVQKC7LU6D7MOSLBDKFHY7YSO3:/var/lib/docker/overlay2/l/R4GGX3SFPMLXTNM3WKMVOKDTOY:/var/lib/docker/overlay2/l/VHGYTU73JLTRCGX45ZF2VGW4FK,upperdir=/var/lib/docker/overlay2/e1fab975d5ffd51474b11a964c82c3bfda1c0e82aec6845a1f12c8150bf61419/diff,workdir=/var/lib/docker/overlay2/e1fab975d5ffd51474b11a964c82c3bfda1c0e82aec6845a1f12c8150bf61419/work,index=off
706 705 0:105 / /proc rw,nosuid,nodev,noexec,relatime - proc proc rw
707 705 0:106 / /dev rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755,inode64
708 707 0:107 / /dev/pts rw,nosuid,noexec,relatime - devpts devpts rw,gid=5,mode=620,ptmxmode=666
709 705 0:108 / /sys ro,nosuid,nodev,noexec,relatime - sysfs sysfs ro
710 709 0:25 / /sys/fs/cgroup ro,nosuid,nodev,noexec,relatime - cgroup2 cgroup rw,nsdelegate,memory_recursiveprot
711 707 0:104 / /dev/mqueue rw,nosuid,nodev,noexec,relatime - mqueue mqueue rw
712 707 0:109 / /dev/shm rw,nosuid,nodev,noexec,relatime - tmpfs shm rw,size=65536k,inode64
713 705 8:3 /var/lib/docker/containers/%[1]v/resolv.conf /etc/resolv.conf rw,relatime - ext4 /dev/sda3 rw
714 705 8:3 /var/lib/docker/containers/%[1]v/hostname /etc/hostname rw,relatime - ext4 /dev/sda3 rw
715 705 8:3 /var/lib/docker/containers/%[1]v/hosts /etc/hosts rw,relatime - ext4 /dev/sda3 rw
716 705 8:3 /var/lib/docker/volumes/ca8074e1a2eb12edc86c59c5108bb48c31bb7ace4b90beb0da8137a9baa45812/_data /etc/nginx/certs rw,relatime master:1 - ext4 /dev/sda3 rw
717 705 8:3 /var/lib/docker/volumes/2cf8a52c907469a56f6e2cc7d1959d74a4dd04131e7edcd53eaf909db28f770f/_data /etc/nginx/dhparam rw,relatime master:1 - ext4 /dev/sda3 rw
662 707 0:107 /0 /dev/console rw,nosuid,noexec,relatime - devpts devpts rw,gid=5,mode=620,ptmxmode=666
663 706 0:105 /bus /proc/bus ro,relatime - proc proc rw
664 706 0:105 /fs /proc/fs ro,relatime - proc proc rw
665 706 0:105 /irq /proc/irq ro,relatime - proc proc rw
666 706 0:105 /sys /proc/sys ro,relatime - proc proc rw
667 706 0:105 /sysrq-trigger /proc/sysrq-trigger ro,relatime - proc proc rw
668 706 0:110 / /proc/acpi ro,relatime - tmpfs tmpfs ro,inode64
669 706 0:106 /null /proc/kcore rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755,inode64
670 706 0:106 /null /proc/keys rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755,inode64
671 706 0:106 /null /proc/latency_stats rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755,inode64
672 706 0:106 /null /proc/timer_list rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755,inode64
673 706 0:106 /null /proc/sched_debug rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755,inode64
674 706 0:111 / /proc/scsi ro,relatime - tmpfs tmpfs ro,inode64
675 709 0:112 / /sys/firmware ro,relatime - tmpfs tmpfs ro,inode64`, ids[2]),
	}

	keys := []string{
		"cpuset",
		"cgroup",
		"mountinfo",
	}

	var filepaths []string
	// Create temporary files with test content
	for _, key := range keys {
		file, err := ioutil.TempFile("", key)
		if err != nil {
			log.Fatal(err)
		}
		defer os.Remove(file.Name())
		if _, err = file.WriteString(contents[key]); err != nil {
			log.Fatal(err)
		}
		filepaths = append(filepaths, file.Name())
	}

	// Each time the HOSTNAME is set to a short form ID, GetCurrentContainerID() should match and return the corresponding full ID
	for _, id := range ids {
		os.Setenv("HOSTNAME", id[0:12])
		if got, exp := GetCurrentContainerID(filepaths...), id; got != exp {
			t.Fatalf("id mismatch with HOSTNAME %v: got %v, exp %v", id[0:12], got, exp)
		}
	}

	// If the Hostname isn't a short form ID, we should match the first valid ID (64 character hex string) instead
	os.Setenv("HOSTNAME", "customhostname")
	if got, exp := GetCurrentContainerID(filepaths...), ids[0]; got != exp {
		t.Fatalf("id mismatch with custom HOSTNAME: got %v, exp %v", got, exp)
	}
}
