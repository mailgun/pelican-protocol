package pelican

import (
	"fmt"
	"os/exec"
	"strings"
)

func StartDockerImage(image string) {
	err := exec.Command("/usr/bin/docker", "run", image, "/sbin/my_init").Start()
	if err != nil {
		//fmt.Fprintf(os.Stderr, "err '%s' in StartDockerImage(). Out: '%s'", err, string(out))
		panic(err)
	}
	fmt.Printf("StartDockerImage() done.\n")
}

func SshAsRootIntoDocker(cmd []string) ([]byte, error) {
	return exec.Command("make", fmt.Sprintf("ARGS='%s'", strings.Join(cmd, " ")), "sshroot").CombinedOutput()
}

func TrimRightNewline(slice []byte) []byte {
	n := len(slice)
	if n > 0 {
		slice = slice[:n-1]
	}
	return slice
}

func RunningDockerId() ([]byte, error) {
	out, err := exec.Command("/usr/bin/docker", "ps", "-q", "-n=1", "-f", "status=running").CombinedOutput()
	out = TrimRightNewline(out)
	return out, err
}

func StopAllDockers() {
	for {
		out, err := RunningDockerId()
		if err != nil {
			panic(err)
		}
		if len(out) == 0 {
			return
		}
		fmt.Printf("StopAllDockers() is stopping '%s'\n", string(out))
		_, err = exec.Command("/usr/bin/docker", "stop", string(out)).CombinedOutput()
		if err != nil {
			panic(err)
		}
	}
}
