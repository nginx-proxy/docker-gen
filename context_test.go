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
		`13:name=systemd:/docker-ce/docker/18862cabc2e0d24142cf93c46ccb6e070c2ea7b996c81c0311ec0309abcbcdfb
12:pids:/docker-ce/docker/18862cabc2e0d24142cf93c46ccb6e070c2ea7b996c81c0311ec0309abcbcdfb
11:hugetlb:/docker-ce/docker/18862cabc2e0d24142cf93c46ccb6e070c2ea7b996c81c0311ec0309abcbcdfb
10:net_prio:/docker-ce/docker/18862cabc2e0d24142cf93c46ccb6e070c2ea7b996c81c0311ec0309abcbcdfb
9:perf_event:/docker-ce/docker/18862cabc2e0d24142cf93c46ccb6e070c2ea7b996c81c0311ec0309abcbcdfb
8:net_cls:/docker-ce/docker/18862cabc2e0d24142cf93c46ccb6e070c2ea7b996c81c0311ec0309abcbcdfb
7:freezer:/docker-ce/docker/18862cabc2e0d24142cf93c46ccb6e070c2ea7b996c81c0311ec0309abcbcdfb
6:devices:/docker-ce/docker/18862cabc2e0d24142cf93c46ccb6e070c2ea7b996c81c0311ec0309abcbcdfb
5:memory:/docker-ce/docker/18862cabc2e0d24142cf93c46ccb6e070c2ea7b996c81c0311ec0309abcbcdfb
4:blkio:/docker-ce/docker/18862cabc2e0d24142cf93c46ccb6e070c2ea7b996c81c0311ec0309abcbcdfb
3:cpuacct:/docker-ce/docker/18862cabc2e0d24142cf93c46ccb6e070c2ea7b996c81c0311ec0309abcbcdfb
2:cpu:/docker-ce/docker/18862cabc2e0d24142cf93c46ccb6e070c2ea7b996c81c0311ec0309abcbcdfb
1:cpuset:/docker-ce/docker/18862cabc2e0d24142cf93c46ccb6e070c2ea7b996c81c0311ec0309abcbcdfb`

	if got, exp := matchDockerCurrentContainerID(cgroup), "18862cabc2e0d24142cf93c46ccb6e070c2ea7b996c81c0311ec0309abcbcdfb"; got != exp {
		t.Fatalf("id mismatch: got %v, exp %v", got, exp)
	}

}
