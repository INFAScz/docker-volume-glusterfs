package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/docker/go-plugins-helpers/volume"
	"github.com/sirupsen/logrus"

	"github.com/origin-nexus/docker-volume-glusterfs/glusterfs-volume"
)

const socketAddress = "/run/docker/plugins/glusterblockfile.sock"

func NewDriver(root string) (*Driver, error) {
	logrus.WithField("method", "new glusterfs driver").Debug(root)

	options := map[string]string{}

	options_str := os.Getenv("OPTIONS")
	for _, option := range strings.Split(options_str, " ") {
		if option == "" {
			continue
		}
		kv := strings.SplitN(option, "=", 2)
		if len(kv) == 1 {
			kv = append(kv, "")
		}
		switch kv[0] {
		default:
			if err := glusterfsvolume.CheckOption(kv[0], kv[1]); err != nil {
				return nil, err
			}
			options[kv[0]] = kv[1]
		}
	}

	loglevel := os.Getenv("LOGLEVEL")
	switch loglevel {
	case "TRACE":
		logrus.SetLevel(logrus.TraceLevel)
	case "DEBUG":
		logrus.SetLevel(logrus.DebugLevel)
	case "INFO":
		logrus.SetLevel(logrus.InfoLevel)
	case "":
		loglevel = "WARNING"
		fallthrough
	case "WARNING":
		logrus.SetLevel(logrus.WarnLevel)
	case "ERROR":
		logrus.SetLevel(logrus.ErrorLevel)
	case "CRITICAL":
		logrus.SetLevel(logrus.ErrorLevel)
	case "NONE":
		logrus.SetLevel(logrus.ErrorLevel)
	default:
		return nil, fmt.Errorf("unknown log level '%v'", loglevel)
	}

	servers := os.Getenv("SERVERS")
	volumeName := os.Getenv("VOLUME_NAME")

	_, dedicatedMounts := options["dedicated-mount"]
	delete(options, "dedicated-mount")

	filesystem, _ := options["filesystem"]
	delete(options, "filesystem")

	filenameFormat, _ := options["filename-format"]
	delete(options, "filename-format")

	size, _ := options["default-size"]
	delete(options, "default-size")

	return &Driver{
		root:      root,
		statePath: filepath.Join(root, "gluster-block-file-state.json"),
		glusterConfig: glusterfsvolume.Config{
			Servers:        servers,
			VolumeName:     volumeName,
			DedicatedMount: dedicatedMounts,
			Options:        options,
		},
		blockFileConfig: BlockFileConfig{
			filenameFormat: filenameFormat,
			filesystem:     filesystem,
			size:           size,
		},
		state: State{
			GlusterBlockVolumes: map[string]*GlusterBlockVolume{},
			GlusterVolumes:      glusterfsvolume.State{},
		},
	}, nil
}

func main() {
	d, err := NewDriver("/data")
	if err != nil {
		logrus.Fatal(err)
	}
	h := volume.NewHandler(d)
	logrus.Infof("listening on %s", socketAddress)
	logrus.Error(h.ServeUnix(socketAddress, 0))
}

func executeCommand(cmd string, args ...string) ([]byte, error) {
	return exec.Command(cmd, args...).CombinedOutput()
}
