package docker

import (
	"errors"
	"github.com/David-Antunes/gone/internal/proxy"
	"log"
	"os"
	"os/exec"
	"strings"
)

var dockerLog = log.New(os.Stdout, "EMULATION INFO: ", log.Ltime)

type dockerNode struct {
	machineId string
	id        string
	mac       string
	ip        string
}

type DockerManager struct {
	machineId string
	proxy     *proxy.ProxyServer
	nodes     map[string]dockerNode
}

func CreateDockerManager(id string, proxyServer *proxy.ProxyServer) *DockerManager {
	return &DockerManager{
		machineId: id,
		proxy:     proxyServer,
		nodes:     make(map[string]dockerNode),
	}
}

func (d *DockerManager) GetMachineId() string {
	return d.machineId
}

func (d *DockerManager) RegisterContainer(machineId string, id string, mac string, ip string) error {
	if d.machineId == "" && machineId != "" {
		return errors.New("emulation is running locally")
	} else if _, ok := d.nodes[machineId]; ok {
		return errors.New("container already exists")
	}
	d.nodes[id] = dockerNode{
		machineId: machineId,
		id:        id,
		mac:       mac,
		ip:        ip,
	}
	return nil
}

// return uniqId, macAddress, ipAddr, nil
func (d *DockerManager) ExecContainer(dockerCmd string) (string, string, string, error) {

	cmd := strings.Split(dockerCmd, " ")
	shell := exec.Command(cmd[0], cmd[1:]...)
	out, err := shell.Output()
	if err != nil {
		dockerLog.Println("Could not execute docker run.", err)
		return "", "", "", err
	}
	containerId := strings.Trim(string(out), " ")
	containerId = strings.Trim(containerId, "\n")

	shell = exec.Command("docker", "pause", containerId)
	_, err = shell.Output()

	if err != nil {
		dockerLog.Println("Could not pause docker container", err)
		shell = exec.Command("docker", "kill", containerId)
		_, err = shell.Output()
		return "", "", "", err
	}

	err = d.proxy.Refresh()
	if err != nil {
		dockerLog.Println("Could not refresh proxy", err)
		//return "", "", "", err
	}

	shell = exec.Command("docker", "inspect", containerId, "--format", "{{.NetworkSettings.Networks.net.MacAddress}}")
	out, err = shell.Output()
	if err != nil {
		dockerLog.Println("Could not fetch container Mac Address", err)
		ClearContainer(containerId)
		return "", "", "", err
	}
	macAddress := strings.Trim(string(out), " ")
	macAddress = strings.Trim(macAddress, "\n")

	shell = exec.Command("docker", "inspect", containerId, "--format", "{{.NetworkSettings.Networks.net.IPAddress}}")
	out, err = shell.Output()
	if err != nil {
		dockerLog.Println("Could not fetch container Ip Address", err)
		ClearContainer(containerId)
		return "", "", "", err
	}

	ipAddr := strings.Trim(string(out), " ")
	ipAddr = strings.Trim(ipAddr, "\n")

	shell = exec.Command("docker", "inspect", containerId, "--format", "{{.Name}}")
	out, err = shell.Output()
	if err != nil {
		dockerLog.Println("Could not fetch container Mac Address", err)
		ClearContainer(containerId)
		return "", "", "", err
	}

	uniqId := strings.Trim(string(out), " ")
	uniqId = strings.Trim(uniqId, "\n")
	uniqId = strings.Trim(uniqId, "/")
	return uniqId, macAddress, ipAddr, nil
}

func ClearContainer(containerId string) {
	exec.Command("docker", "unpause", containerId).Output()
	exec.Command("docker", "kill", containerId).Output()
}

func (d *DockerManager) PropagateArp(ip string, mac string) error {

	for _, node := range d.nodes {
		if node.machineId != d.GetMachineId() {
			continue
		}
		if node.ip == ip && node.mac == mac {
			continue
		}

		shell := exec.Command("docker", "inspect", node.id, "--format", "{{.State.Pid}}")
		out, err := shell.Output()
		if err != nil {
			dockerLog.Println("Could not fetch namespace id", err)
			ClearContainer(node.id)
			return err
		}
		pid := strings.Trim(string(out), " ")
		pid = strings.Trim(pid, "\n")

		out, err = exec.Command("nsenter", "--target", pid, "--net", "arp", "-s", ip, mac).Output()
		if err != nil {
			dockerLog.Println(err)
			dockerLog.Println(string(out))
		}
	}

	return nil
}

func (d *DockerManager) RemoveArp(ip string) error {

	for _, node := range d.nodes {
		if node.machineId != d.GetMachineId() {
			continue
		}

		shell := exec.Command("docker", "inspect", node.id, "--format", "{{.State.Pid}}")
		out, err := shell.Output()
		if err != nil {
			dockerLog.Println("Could not fetch namespace id", err)
			ClearContainer(node.id)
			return err
		}
		pid := strings.Trim(string(out), " ")
		pid = strings.Trim(pid, "\n")

		out, err = exec.Command("nsenter", "--target", pid, "--net", "arp", "-d", ip).Output()
		if err != nil {
			dockerLog.Println(err)
			dockerLog.Println(string(out))
			continue
		}
	}
	return nil
}

func (d *DockerManager) BootstrapContainer(id string) error {

	shell := exec.Command("docker", "inspect", id, "--format", "{{.State.Pid}}")
	out, err := shell.Output()
	if err != nil {
		dockerLog.Println("Could not fetch namespace id", err)
		return err
	}

	pid := strings.Trim(string(out), " ")
	pid = strings.Trim(pid, "\n")
	o, err := exec.Command("nsenter", "--target", pid, "--net", "ethtool", "-K", "eth0", "rx", "off", "tx", "off").Output()
	if err != nil {
		dockerLog.Println(o)
		dockerLog.Println("Failed ethtool", err)
		ClearContainer(id)
		return err
	}

	_, _ = exec.Command("nsenter", "--target", pid, "--net", "ping", "-b", "-w", "1", "-c", "1", "10.1.0.1").Output()
	//if err != nil {
	//	dockerLog.Println(o)
	//	dockerLog.Println("failed ping", err)
	//	ClearContainer(id)
	//	return err
	//}

	for _, node := range d.nodes {
		if node.id == id {
			continue
		}
		out, err = exec.Command("nsenter", "--target", pid, "--net", "arp", "-s", node.ip, node.mac).Output()
		if err != nil {
			dockerLog.Println(err)
			dockerLog.Println(string(out))
			continue
		}
	}

	shell = exec.Command("docker", "unpause", id)
	_, err = shell.Output()

	if err != nil {
		dockerLog.Println("Could not unpause container", err)
		ClearContainer(id)
		return err
	}
	return nil
}

func (d *DockerManager) RemoveNode(id string) error {
	container, ok := d.nodes[id]
	if ok {
		delete(d.nodes, id)
	} else {
		return errors.New("container not found")
	}
	if container.machineId == d.machineId {

		shell := exec.Command("docker", "kill", id)
		_, err := shell.Output()
		if err != nil {
			dockerLog.Println("could not kill container", err)
			return err
		}
		shell = exec.Command("docker", "rm", id)
		_, err = shell.Output()
		if err != nil {
			dockerLog.Println("could not remove container", err)
			return err
		}
	}
	err := d.RemoveArp(container.ip)
	if err != nil {
		return err
	}
	return nil
}
