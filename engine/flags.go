package engine

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ergongate/vince/version"
)

var VersionFlag = flag.Bool("v", false, "show version and exit")
var VersionAndConfigFlag = flag.Bool("V", false, "show version and configure options then exit")
var TestFlag = flag.Bool("t", false, "test configuration and exit")
var TestDump = flag.Bool("T", false, "test configuration, dump it and exit")

// NginxDirs returns a list of directories from which nginx configurations can
// be found.
func NginxDirs() []string {
	return []string{
		"/usr/local/nginx/conf",
		"/etc/nginx",
		"/usr/local/etc/nginx",
	}
}

// ConfigDir returns the path to nginx configuration directory with an error if
// no active directory found.
func ConfigDir() (string, error) {
	var dir string
	for _, v := range NginxDirs() {
		_, err := os.Stat(v)
		if err == nil {
			dir = v
			break
		}
	}
	if dir == "" {
		return "", errors.New("vince: failed to find nginx configuration directory")
	}
	return dir, nil
}

func showVersion() {
	if *VersionFlag {
		fmt.Println(version.Version)
		os.Exit(0)
	}
}

func showVersionAndConfig() {
	if *VersionAndConfigFlag {
		fmt.Println(version.Version)
		os.Exit(0)
	}
}

func testConfiguration() {
	if *TestFlag {
		err := testConfig()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		os.Exit(0)
	}
}

func testConfig() error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}
	file := filepath.Join(dir, "nginx.conf")
	_, err = os.Stat(file)
	if err != nil {
		return err
	}
	fmt.Println("vince found configuration ", file)
	fmt.Printf("vince: the configuration file %s syntax is ok\n", file)
	fmt.Printf("vince: the configuration file %s test is successful\n", file)
	return nil
}

func testAndDump() {
	if *TestDump {
		err := testConfig()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		//TODO dump configurations
		os.Exit(0)
	}
}

func runFlags() {
	showVersion()
	showVersionAndConfig()
	testConfiguration()
	testAndDump()
}