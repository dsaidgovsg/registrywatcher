package testutils

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/term"
	"github.com/docker/go-connections/nat"
	"github.com/dsaidgovsg/registrywatcher/utils"
	"github.com/spf13/viper"
)

// TestHelper implements methods to manipulate docker registry from test cases
type TestHelper struct {
	DockerClient *client.Client
	Conf         *viper.Viper
}

func NewTestHelper(conf *viper.Viper) *TestHelper {
	dcli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(fmt.Errorf("could not connect to docker: %v", err))
	}

	helper := TestHelper{
		Conf:         conf,
		DockerClient: dcli,
	}

	return &helper
}

// StartPostgres starts a new postgres container.
func (helper *TestHelper) StartPostgres() (string, error) {
	port, err := nat.NewPort("tcp", helper.Conf.GetString("postgres_container_port"))
	if err != nil {
		return "", err
	}

	image := helper.Conf.GetString("postgres_container_image")
	if err := helper.pullDockerImage(image); err != nil {
		return "", err
	}

	c, err := helper.DockerClient.ContainerCreate(
		context.Background(),
		&container.Config{
			Image: image,
			ExposedPorts: map[nat.Port]struct{}{
				port: {},
			},
			Env: []string{
				"POSTGRES_PASSWORD=registry-watcher",
				"POSTGRES_DB=registry-watcher",
				"POSTGRES_USER=registry-watcher",
			},
		},
		&container.HostConfig{
			PortBindings: map[nat.Port][]nat.PortBinding{
				port: []nat.PortBinding{{
					HostIP:   "0.0.0.0",
					HostPort: port.Port(),
				}},
			},
			NetworkMode: "bridge",
			RestartPolicy: container.RestartPolicy{
				Name: "always",
			},
		},
		nil, "")
	if err != nil {
		return "", err
	}

	// start the container
	if err := helper.DockerClient.ContainerStart(context.Background(), c.ID, types.ContainerStartOptions{}); err != nil {

		// Try 4 more times
		// 5, 10, 20, 40
		for i := 0; i < 4 && err != nil; i++ {
			time.Sleep(time.Duration(5*math.Pow(2, float64(i))) * time.Second)
			err = helper.DockerClient.ContainerStart(context.Background(), c.ID, types.ContainerStartOptions{})
		}
		if err != nil {
			return "", err
		}
	}

	return c.ID, err
}

// StartRegistry starts a new registry container.
func (helper *TestHelper) StartRegistry() (string, string, error) {
	config := "noauth.yml"
	port, err := nat.NewPort("tcp", helper.Conf.GetString("registry_container_port"))
	if err != nil {
		return "", "", err
	}

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", "", errors.New("no caller information")
	}

	image := helper.Conf.GetString("registry_container_image")
	if err := helper.pullDockerImage(image); err != nil {
		return "", "", err
	}

	r, err := helper.DockerClient.ContainerCreate(
		context.Background(),
		&container.Config{
			Image: image,
			ExposedPorts: map[nat.Port]struct{}{
				port: {},
			},
		},
		&container.HostConfig{
			PortBindings: map[nat.Port][]nat.PortBinding{
				port: []nat.PortBinding{{
					HostIP:   "0.0.0.0",
					HostPort: port.Port(),
				}},
			},
			NetworkMode: "bridge",
			Binds: []string{
				filepath.Join(filepath.Dir(filename), "configs", config) + ":" + "/etc/docker/registry/config.yml" + ":ro",
				filepath.Join(filepath.Dir(filename), "configs", "htpasswd") + ":" + "/etc/docker/registry/htpasswd" + ":ro",
				filepath.Join(filepath.Dir(filename), "snakeoil") + ":" + "/etc/docker/registry/ssl" + ":ro",
			},
			RestartPolicy: container.RestartPolicy{
				Name: "always",
			},
		},
		nil, "")
	if err != nil {
		return "", "", err
	}

	if err := helper.DockerClient.ContainerStart(context.Background(), r.ID, types.ContainerStartOptions{}); err != nil {

		// Try 4 more times
		// 5, 10, 20, 40
		for i := 0; i < 4 && err != nil; i++ {
			time.Sleep(time.Duration(5*math.Pow(2, float64(i))) * time.Second)
			err = helper.DockerClient.ContainerStart(context.Background(), r.ID, types.ContainerStartOptions{})
		}
		if err != nil {
			return "", "", err
		}
	}

	addr := "https://localhost:" + port.Port()

	if err := helper.waitForConn(addr, filepath.Join(filepath.Dir(filename), "snakeoil", "cert.pem"), filepath.Join(filepath.Dir(filename), "snakeoil", "key.pem")); err != nil {
		return r.ID, addr, err
	}

	if err := helper.dockerLogin("localhost:" + port.Port()); err != nil {
		return r.ID, addr, err
	}

	// Add image to the registry.
	publicImageName := helper.Conf.GetString("base_public_image")
	testRepoName := helper.Conf.GetStringSlice("watched_repositories")[0]
	_, registryDomain, registryPrefix, _ := utils.ExtractRegistryInfo(helper.Conf, testRepoName)
	mockImageName := utils.ConstructImageName(registryDomain, registryPrefix, testRepoName, "v0.0.1")
	if err := helper.AddImageToRegistry(publicImageName, mockImageName); err != nil {
		return r.ID, addr, err
	}

	return r.ID, addr, nil
}

// RemoveContainer removes with force a container by it's container ID.
func (helper *TestHelper) RemoveContainer(ctrs ...string) (err error) {
	for _, c := range ctrs {
		err = helper.DockerClient.ContainerRemove(context.Background(), c,
			types.ContainerRemoveOptions{
				RemoveVolumes: true,
				Force:         true,
			})
	}

	return err
}

// dockerLogin logins via the command line to a docker registry
func (helper *TestHelper) dockerLogin(addr string) error {
	_, _, _, registryAuth := utils.ExtractRegistryInfo(helper.Conf, "testrepo")
	username, password, err := utils.DecodeAuthString(registryAuth)
	cmd := exec.Command("docker", "login", "--username", username, "--password", password, addr)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker login [%s] failed with output %q and error: %v", strings.Join(cmd.Args, " "), string(out), err)
	}
	return nil
}

// AddImageToRegistry adds images to a registry.
func (helper *TestHelper) AddImageToRegistry(publicImage, mockImage string) error {
	_, _, _, registryAuth := utils.ExtractRegistryInfo(helper.Conf, "testrepo")
	username, password, err := utils.DecodeAuthString(registryAuth)

	if err := helper.pullDockerImage(publicImage); err != nil {
		return err
	}

	if err := helper.DockerClient.ImageTag(context.Background(), publicImage, mockImage); err != nil {
		return err
	}

	auth, err := ConstructRegistryAuth(username, password)
	if err != nil {
		return err
	}

	resp, err := helper.DockerClient.ImagePush(context.Background(), mockImage, types.ImagePushOptions{
		RegistryAuth: auth,
	})
	if err != nil {
		return err
	}
	defer resp.Close()

	fd, isTerm := term.GetFdInfo(os.Stdout)

	return jsonmessage.DisplayJSONMessagesStream(resp, os.Stdout, fd, isTerm, nil)
}

func (helper *TestHelper) pullDockerImage(image string) error {
	exists, err := helper.imageExists(image)
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	resp, err := helper.DockerClient.ImagePull(context.Background(), image, types.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer resp.Close()

	fd, isTerm := term.GetFdInfo(os.Stdout)

	return jsonmessage.DisplayJSONMessagesStream(resp, os.Stdout, fd, isTerm, nil)
}

func (helper *TestHelper) imageExists(image string) (bool, error) {
	_, _, err := helper.DockerClient.ImageInspectWithRaw(context.Background(), image)
	if err == nil {
		return true, nil
	}

	if client.IsErrNotFound(err) {
		return false, nil
	}

	return false, err
}

// waitForConn takes a tcp addr and waits until it is reachable
func (helper *TestHelper) waitForConn(addr, cert, key string) error {
	tlsCert, err := tls.LoadX509KeyPair(cert, key)
	if err != nil {
		return fmt.Errorf("could not load X509 key pair: %v. Make sure the key is not encrypted", err)
	}

	certPool, err := x509.SystemCertPool()
	if err != nil {
		return fmt.Errorf("failed to read system certificates: %v", err)
	}
	pem, err := ioutil.ReadFile(cert)
	if err != nil {
		return fmt.Errorf("could not read CA certificate %s: %v", cert, err)
	}
	if !certPool.AppendCertsFromPEM(pem) {
		return fmt.Errorf("failed to append certificates from PEM file: %s", cert)
	}

	c := http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			TLSClientConfig: &tls.Config{
				Certificates: []tls.Certificate{tlsCert},
				MinVersion:   tls.VersionTLS12,
				CipherSuites: []uint16{
					tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
					tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				},
				RootCAs: certPool,
			},
		},
	}

	n := 0
	max := 10
	for n < max {
		if _, err := c.Get(addr + "/v2/"); err != nil {
			fmt.Printf("try number %d to %s: %v\n", n, addr, err)
			n++
			if n != max {
				fmt.Println("sleeping for 1 second then will try again...")
				time.Sleep(time.Second)
			} else {
				return fmt.Errorf("[WHOOPS]: maximum retries for %s exceeded", addr)
			}
			continue
		} else {
			break
		}
	}

	return nil
}

// ConstructRegistryAuth serializes the auth configuration as JSON base64 payload.
func ConstructRegistryAuth(identity, secret string) (string, error) {
	buf, err := json.Marshal(types.AuthConfig{Username: identity, Password: secret})
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(buf), nil
}
