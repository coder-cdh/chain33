package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"path"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io"
	//"io/ioutil"
	"path/filepath"
	"strings"
	"strconv"
)

type ScpInfo struct {
	userName      string
	passWord      string
	hostIp        string
	port          int
	localFilePath string
	remoteDir     string
}
type CmdInfo struct {
	userName string
	passWord string
	hostIp   string
	port     int
	cmd      string
}

func sshconnect(user, password, host string, port int) (*ssh.Session, error) {
	var (
		auth         []ssh.AuthMethod
		addr         string
		clientConfig *ssh.ClientConfig
		client       *ssh.Client
		session      *ssh.Session
		err          error
	)
	// get auth method
	auth = make([]ssh.AuthMethod, 0)
	auth = append(auth, ssh.Password(password))

	clientConfig = &ssh.ClientConfig{
		User:    user,
		Auth:    auth,
		Timeout: 30 * time.Second,
		//需要验证服务端，不做验证返回nil就可以
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}
	// connet to ssh
	addr = fmt.Sprintf("%s:%d", host, port)
	if client, err = ssh.Dial("tcp", addr, clientConfig); err != nil {
		return nil, err
	}
	// create session
	if session, err = client.NewSession(); err != nil {
		return nil, err
	}
	return session, nil
}
func sftpconnect(user, password, host string, port int) (*sftp.Client, error) {
	var (
		auth         []ssh.AuthMethod
		addr         string
		clientConfig *ssh.ClientConfig
		sshClient    *ssh.Client
		sftpClient   *sftp.Client
		err          error
	)
	// get auth method
	auth = make([]ssh.AuthMethod, 0)
	auth = append(auth, ssh.Password(password))

	clientConfig = &ssh.ClientConfig{
		User:    user,
		Auth:    auth,
		Timeout: 30 * time.Second,
		//需要验证服务端，不做验证返回nil就可以
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}
	// connet to ssh
	addr = fmt.Sprintf("%s:%d", host, port)

	if sshClient, err = ssh.Dial("tcp", addr, clientConfig); err != nil {
		return nil, err
	}
	// create sftp client
	if sftpClient, err = sftp.NewClient(sshClient); err != nil {
		return nil, err
	}
	return sftpClient, nil
}
func ScpFileFromLocalToRemote(si *ScpInfo) {
	sftpClient, err := sftpconnect(si.userName, si.passWord, si.hostIp, si.port)
	if err != nil {
		fmt.Println("sftconnect have a err!")
		log.Fatal(err)
		panic(err)
	}
	defer sftpClient.Close()
	srcFile, err := os.Open(si.localFilePath)
	if err != nil {
		log.Fatal(err)
		panic(err)
	}
	defer srcFile.Close()

	var remoteFileName = path.Base(si.localFilePath)
	fmt.Println("remoteFileName:", remoteFileName)
	dstFile, err := sftpClient.Create(path.Join(si.remoteDir, remoteFileName))
	if err != nil {
		log.Fatal(err)
	}
	defer dstFile.Close()
	//bufReader := bufio.NewReader(srcFile)
	//b := bytes.NewBuffer(make([]byte,0))

	buf := make([]byte, 1024000)
	i := 0
	for {
		//n, err := bufReader.Read(buf)
		n, _ := srcFile.Read(buf)
		if err != nil && err != io.EOF {
			panic(err)
		}
		if n == 0 {
			break
		}
		i++
		fmt.Println("times:==========", i)
		dstFile.Write(buf[0:n])
	}
	fmt.Println("copy file to remote server finished!")
}
func RemoteExec(cmdInfo *CmdInfo)error{
	//A Session only accepts one call to Run, Start or Shell.
	session, err := sshconnect(cmdInfo.userName, cmdInfo.passWord, cmdInfo.hostIp, cmdInfo.port)
	if err != nil {
	   return err
	}
	defer session.Close()
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	err = session.Run(cmdInfo.cmd)
	if err !=nil {
      return err
	}
	return nil
}
func remoteScp(si *ScpInfo, reqnum chan struct{}) {
	defer func() {
		reqnum <- struct{}{}
	}()
	ScpFileFromLocalToRemote(si)
	//session, err := sshconnect("ubuntu", "Fuzamei#123456", "raft15258.chinacloudapp.cn", 22)
	fmt.Println("remote exec cmds.......:")


}
func getCurrentDirectory() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(dir)
	return strings.Replace(dir, "\\", "/", -1)
}
func main() {
	start := time.Now()

	////读取当前目录下的文件
	//dir_list, e := ioutil.ReadDir("D:/Repository/src/code.aliyun.com/chain33/chain33/consensus/drivers/raft/tools/scripts")
	//if e != nil {
	//	fmt.Println("read dir error")
	//	return
	//}
	//for i, v := range dir_list {
	//	fmt.Println(i, "=", v.Name())
	//}

	fmt.Println(getCurrentDirectory())
	scpInfo := &ScpInfo{}
	scpInfo.userName = "ubuntu"
	scpInfo.passWord = "Fuzamei#123456"
	scpInfo.hostIp = "raft15258.chinacloudapp.cn"
	scpInfo.port = 22
	scpInfo.localFilePath = "D:/Repository/src/code.aliyun.com/chain33/chain33/consensus/drivers/raft/tools/scripts/chain33.tgz"
	scpInfo.remoteDir = "/home/ubuntu/deploy"

	cmdInfo:=&CmdInfo{}
	cmdInfo.userName = "ubuntu"
	cmdInfo.passWord = "Fuzamei#123456"
	cmdInfo.hostIp = "raft15258.chinacloudapp.cn"
	cmdInfo.port = 22
	var arr []*ScpInfo
	arr = append(arr, scpInfo)
	// os.Open(scpInfo.localFilePath)
	//多协程启动部署
	reqC := make(chan struct{}, len(arr))
	for _, sc := range arr {
		cmdInfo.cmd="mkdir -p /home/ubuntu/deploy"
		RemoteExec(cmdInfo)
		go remoteScp(sc, reqC)
	}
	for i := 0; i < len(arr); i++ {
		<-reqC
	}
	for i,_ := range arr {
		cmdInfo.cmd="cd /home/ubuntu/deploy;tar -xvf chain33.tgz;bash raft_conf.sh;bash run.sh "+strconv.FormatInt(int64(i)+1,10)
		RemoteExec(cmdInfo)
	}
	timeCommon := time.Now()
	log.Printf("read common cost time %v\n", timeCommon.Sub(start))
}
