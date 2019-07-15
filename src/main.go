package main

import "errors"
import "fmt"
import "io"
import "net/http"
import "os"
import "os/exec"
import "os/signal"
import "path/filepath"
import "runtime"
import "strconv"
import "syscall"

import "github.com/bndr/gojenkins"


func removeWorkdirContents(dir string) (error) {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}

	defer d.Close()

	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}

	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}

	return nil
}


type slave struct {
	executors int
	ip string
	port string
	name string
	workdir string
	user string
	passwd string
	exec string
	cmd *exec.Cmd
	jenkins *gojenkins.Jenkins
}


func (s *slave) prepareWorkDir() (error) {
	var err error

	_, err = os.Stat(s.workdir)
	if err != nil {
		fmt.Printf("create workdir [%s]\n", s.workdir)
		err = os.MkdirAll(s.workdir, 0777)
		if err != nil {
			return fmt.Errorf("fail create workdir [%s]", err)
		}
	} else {
		err = removeWorkdirContents(s.workdir)
		if err != nil {
			return fmt.Errorf("fail empty workdir [%s]", err)
		}
	}

	err = os.Chdir(s.workdir)
	if err != nil {
		return fmt.Errorf("fail change workdir [%s]", err)
	}

	return nil
}


func (s *slave) getName() {
	s.name = os.Getenv("SLAVE_NAME")
	if len(s.name) > 0 {
		return
	}

	if runtime.GOOS == "linux" {
		host := os.Getenv("HOSTNAME")
		s.name = "docker-slave-" + host
	} else if runtime.GOOS == "darwin" {
		s.name = "docker-slave-mac"
	} else if runtime.GOOS == "windows" {
		s.name = "docker-slave-windows"
	}
}


func (s *slave) initSlave() (error) {
	var err error

	s.getName()

	s.ip = os.Getenv("SLAVE_IP")
	if len(s.ip) == 0 {
		s.ip = "localhost"
	}

	s.port = os.Getenv("SLAVE_PORT")
	if len(s.port) == 0 {
		s.port = "8080"
	}

	s.user = os.Getenv("SLAVE_USER")
	if len(s.user) == 0 {
		s.user = "admin"
	}

	s.passwd = os.Getenv("SLAVE_PASSWD")
	if len(s.passwd) == 0 {
		s.passwd = "admin123"
	}

	s.executors, err = strconv.Atoi(os.Getenv("SLAVE_EXECUTORS"))
	if err != nil {
		s.executors = 1
	}

	s.workdir = os.Getenv("SLAVE_WORKDIR")
	if len(s.workdir) == 0 {
		s.workdir = "/var/lib/jenkins"
	}

	err = s.prepareWorkDir()
	if err != nil {
		return err
	}

	fmt.Printf("SLAVE_NAME %s\n", s.name)
	fmt.Printf("SLAVE_IP %s\n", s.ip)
	fmt.Printf("SLAVE_PORT %s\n", s.port)
	fmt.Printf("SLAVE_USER %s\n", s.user)
	fmt.Printf("SLAVE_PASSWD %s\n", s.passwd)
	fmt.Printf("SLAVE_EXECUTORS %d\n", s.executors)
	fmt.Printf("SLAVE_WORKDIR %s\n", s.workdir)

	return err
}


func (s *slave) getSlaveBin() (error) {
	var slave_jar string
	var url string

	slave_jar = "/var/lib/jenkins/slave.jar"
	url = "http://" + s.ip + ":" + s.port + "/jnlpJars/slave.jar"

	_, err := os.Stat(slave_jar)
	if err == nil {
		fmt.Printf("removing previous slave.jar\n")
		os.Remove(slave_jar)
	}

	out, err := os.Create(slave_jar)
	if err != nil {
		return errors.New("create local slave.jar")
	}

	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return errors.New("http get slave.jar")
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http status %s", resp.Status)
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return errors.New("copy local slave.jar")
	}

	s.exec = slave_jar

	return nil
}


func (s *slave) registerSlave() (error) {
	var err error
	var url string

	url = "http://" + s.ip + ":" + s.port

	s.jenkins = gojenkins.CreateJenkins(nil, url, s.user, s.passwd)
	_, err = s.jenkins.Init()
	if err != nil {
		return fmt.Errorf("fail create jenkins context [%s]", err)
	}

	// TODO provide all slave info
	node1, err := s.jenkins.CreateNode(s.name, s.executors, "Node 1 Description", s.workdir,
					"", map[string]string{"method": "JNLPLauncher"})
	if err != nil {
		return fmt.Errorf("fail create jenkins node [%s]\n", err)
	}

	fmt.Printf("created node [%s]\n", node1.GetName())

	return nil
}


func (s *slave) deregisterSlave() (error) {
	var err error

	_, err = s.jenkins.DeleteNode(s.name)
	if err != nil {
		return fmt.Errorf("fail delete node [%s]", err)
	}

	fmt.Printf("deleted node [%s]\n", s.name)

	return nil
}


func (s *slave) startSlave() (error) {
	var err error
	var bin string
	var args []string
	var url string

	bin = "/usr/bin/java"
	url = "http://" + s.ip + ":" + s.port + "/computer/" + s.name + "/slave-agent.jnlp"
	args = append(args, "-jar", s.exec, "-noReconnect", "-jnlpUrl", url)
	args = append(args, "-jnlpCredentials")
	args = append(args, s.user + ":" + s.passwd)

	fmt.Printf("bin [%s], args %v\n", bin, args)

	cmd := exec.Command(bin, args...)
	cmd.Stdin = nil
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("fail cmd [%s]", err)
	}

	s.cmd = cmd

	return nil
}


func (s *slave) stopSlave() (error){
	var err error

	fmt.Printf("kill slave [%s]\n", s.name)

	err = s.cmd.Process.Kill()
	if err != nil {
		return fmt.Errorf("fail kill cmd [%s]", err)
	}

	return nil
}


func (s *slave) waitSignal(done_ch chan bool) {
	sigc := make(chan os.Signal, 1)

	signal.Notify(sigc, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	for {
		select {
		case s := <-sigc:
			fmt.Printf("signal [%s] received, shutting down\n", s)
			done_ch <- false
		}
	}
}


func (s *slave) waitSlave(done_ch chan bool) {
	var err error

	err = s.cmd.Wait()
	if err != nil {
		fmt.Printf("slave program exited with error [%v]\n", err)
	} else {
		fmt.Printf("slave program exited successfully\n")
	}

	done_ch <- true
}


func (s *slave) runSlave() (error) {
	var err error
	var exited bool
	var done_ch chan bool

	err = s.startSlave()
	if err != nil {
		return err
	}

	done_ch = make(chan bool)

	go s.waitSignal(done_ch)
	go s.waitSlave(done_ch)

	exited = <-done_ch
	if exited {
		return nil
	}

	err = s.stopSlave()
	if err != nil {
		return err
	}

	return nil
}


func main() {
	var s slave
	var err error

	s.initSlave()

	err = s.getSlaveBin()
	if err != nil {
		fmt.Printf("error: [%s]\n", err)
		panic("get bin: something went wrong")
	}

	err = s.registerSlave()
	if err != nil {
		fmt.Printf("error: [%s]\n", err)
		panic("register: something went wrong")
	}

	err = s.runSlave()
	if err != nil {
		fmt.Printf("error: [%s]\n", err)
		panic("run slave: something went wrong")
	}

	err = s.deregisterSlave()
	if err != nil {
		fmt.Printf("error: [%s]\n", err)
		panic("deregister: something went wrong")
	}
}
