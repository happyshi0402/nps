package common

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"github.com/cnlh/nps/lib/crypt"
	"github.com/cnlh/nps/lib/lg"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
)

//Judging Compression Mode
func GetCompressType(compress string) (int, int) {
	switch compress {
	case "":
		return COMPRESS_NONE_DECODE, COMPRESS_NONE_ENCODE
	case "snappy":
		return COMPRESS_SNAPY_DECODE, COMPRESS_SNAPY_ENCODE
	default:
		lg.Fatalln("数据压缩格式错误")
	}
	return COMPRESS_NONE_DECODE, COMPRESS_NONE_ENCODE
}

//Get the corresponding IP address through domain name
func GetHostByName(hostname string) string {
	if !DomainCheck(hostname) {
		return hostname
	}
	ips, _ := net.LookupIP(hostname)
	if ips != nil {
		for _, v := range ips {
			if v.To4() != nil {
				return v.String()
			}
		}
	}
	return ""
}

//Check the legality of domain
func DomainCheck(domain string) bool {
	var match bool
	IsLine := "^((http://)|(https://))?([a-zA-Z0-9]([a-zA-Z0-9\\-]{0,61}[a-zA-Z0-9])?\\.)+[a-zA-Z]{2,6}(/)"
	NotLine := "^((http://)|(https://))?([a-zA-Z0-9]([a-zA-Z0-9\\-]{0,61}[a-zA-Z0-9])?\\.)+[a-zA-Z]{2,6}"
	match, _ = regexp.MatchString(IsLine, domain)
	if !match {
		match, _ = regexp.MatchString(NotLine, domain)
	}
	return match
}

//Check if the Request request is validated
func CheckAuth(r *http.Request, user, passwd string) bool {
	s := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
	if len(s) != 2 {
		return false
	}

	b, err := base64.StdEncoding.DecodeString(s[1])
	if err != nil {
		return false
	}

	pair := strings.SplitN(string(b), ":", 2)
	if len(pair) != 2 {
		return false
	}
	return pair[0] == user && pair[1] == passwd
}

//get bool by str
func GetBoolByStr(s string) bool {
	switch s {
	case "1", "true":
		return true
	}
	return false
}

//get str by bool
func GetStrByBool(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

//int
func GetIntNoErrByStr(str string) int {
	i, _ := strconv.Atoi(str)
	return i
}

//Get verify value
func Getverifyval(vkey string) string {
	return crypt.Md5(vkey)
}

//Change headers and host of request
func ChangeHostAndHeader(r *http.Request, host string, header string, addr string) {
	if host != "" {
		r.Host = host
	}
	if header != "" {
		h := strings.Split(header, "\n")
		for _, v := range h {
			hd := strings.Split(v, ":")
			if len(hd) == 2 {
				r.Header.Set(hd[0], hd[1])
			}
		}
	}
	addr = strings.Split(addr, ":")[0]
	r.Header.Set("X-Forwarded-For", addr)
	r.Header.Set("X-Real-IP", addr)
}

//Read file content by file path
func ReadAllFromFile(filePath string) ([]byte, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(f)
}

// FileExists reports whether the named file or directory exists.
func FileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

//Judge whether the TCP port can open normally
func TestTcpPort(port int) bool {
	l, err := net.ListenTCP("tcp", &net.TCPAddr{net.ParseIP("0.0.0.0"), port, ""})
	defer l.Close()
	if err != nil {
		return false
	}
	return true
}

//Judge whether the UDP port can open normally
func TestUdpPort(port int) bool {
	l, err := net.ListenUDP("udp", &net.UDPAddr{net.ParseIP("0.0.0.0"), port, ""})
	defer l.Close()
	if err != nil {
		return false
	}
	return true
}

//Write length and individual byte data
//Length prevents sticking
//# Characters are used to separate data
func BinaryWrite(raw *bytes.Buffer, v ...string) {
	buffer := new(bytes.Buffer)
	var l int32
	for _, v := range v {
		l += int32(len([]byte(v))) + int32(len([]byte("#")))
		binary.Write(buffer, binary.LittleEndian, []byte(v))
		binary.Write(buffer, binary.LittleEndian, []byte("#"))
	}
	binary.Write(raw, binary.LittleEndian, buffer.Bytes())
}
