package bridge

import (
	"../lib"
	"errors"
	"net"
	"sync"
	"time"
)

type Client struct {
	tunnel        *lib.Conn
	signal        *lib.Conn
	linkMap       map[int]*lib.Link
	linkStatusMap map[int]bool
	stop          chan bool
	sync.RWMutex
}

type Bridge struct {
	TunnelPort int              //通信隧道端口
	listener   *net.TCPListener //server端监听
	Client     map[int]*Client
	RunList    map[int]interface{} //运行中的任务
	lock       sync.Mutex
	tunnelLock sync.Mutex
	clientLock sync.Mutex
}

func NewTunnel(tunnelPort int, runList map[int]interface{}) *Bridge {
	t := new(Bridge)
	t.TunnelPort = tunnelPort
	t.Client = make(map[int]*Client)
	t.RunList = runList
	return t
}

func (s *Bridge) StartTunnel() error {
	var err error
	s.listener, err = net.ListenTCP("tcp", &net.TCPAddr{net.ParseIP("0.0.0.0"), s.TunnelPort, ""})
	if err != nil {
		return err
	}
	go s.tunnelProcess()
	return nil
}

//tcp server
func (s *Bridge) tunnelProcess() error {
	var err error
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			lib.Println(err)
			continue
		}
		go s.cliProcess(lib.NewConn(conn))
	}
	return err
}

//验证失败，返回错误验证flag，并且关闭连接
func (s *Bridge) verifyError(c *lib.Conn) {
	c.Write([]byte(lib.VERIFY_EER))
	c.Conn.Close()
}

func (s *Bridge) cliProcess(c *lib.Conn) {
	c.SetReadDeadline(5)
	var buf []byte
	var err error
	if buf, err = c.ReadLen(32); err != nil {
		c.Close()
		return
	}
	//验证
	id, err := lib.GetCsvDb().GetIdByVerifyKey(string(buf), c.Conn.RemoteAddr().String())
	if err != nil {
		lib.Println("当前客户端连接校验错误，关闭此客户端:", c.Conn.RemoteAddr())
		s.verifyError(c)
		return
	}
	//做一个判断 添加到对应的channel里面以供使用
	if flag, err := c.ReadFlag(); err == nil {
		s.typeDeal(flag, c, id)
	}
	return
}

func (s *Bridge) closeClient(id int) {
	s.clientLock.Lock()
	defer s.clientLock.Unlock()
	if v, ok := s.Client[id]; ok {
		v.signal.WriteClose()
		delete(s.Client, id)
	}
}

//tcp连接类型区分
func (s *Bridge) typeDeal(typeVal string, c *lib.Conn, id int) {
	switch typeVal {
	case lib.WORK_MAIN:
		//客户端已经存在，下线
		s.clientLock.Lock()
		if _, ok := s.Client[id]; ok {
			s.clientLock.Unlock()
			s.closeClient(id)
		} else {
			s.clientLock.Unlock()
		}
		s.clientLock.Lock()

		s.Client[id] = &Client{
			linkMap:       make(map[int]*lib.Link),
			stop:          make(chan bool),
			linkStatusMap: make(map[int]bool),
		}
		lib.Printf("客户端%d连接成功,地址为：%s", id, c.Conn.RemoteAddr())
		s.Client[id].signal = c
		s.clientLock.Unlock()
		go s.GetStatus(id)
	case lib.WORK_CHAN:
		s.clientLock.Lock()
		if v, ok := s.Client[id]; ok {
			s.clientLock.Unlock()
			v.tunnel = c
		} else {
			s.clientLock.Unlock()
			return
		}
		go s.clientCopy(id)
	}
	c.SetAlive()
	return
}

//等待
func (s *Bridge) waitStatus(clientId, id int) bool {
	ticker := time.NewTicker(time.Millisecond * 100)
	stop := time.After(time.Second * 10)
	for {
		select {
		case <-ticker.C:
			s.clientLock.Lock()
			if v, ok := s.Client[clientId]; ok {
				s.clientLock.Unlock()
				v.Lock()
				if vv, ok := v.linkStatusMap[id]; ok {
					ticker.Stop()
					v.Unlock()
					return vv
				}
				v.Unlock()
			} else {
				s.clientLock.Unlock()
			}
		case <-stop:
			return false
		}
	}
	return false
}

func (s *Bridge) SendLinkInfo(clientId int, link *lib.Link) (tunnel *lib.Conn, err error) {
	s.clientLock.Lock()
	if v, ok := s.Client[clientId]; ok {
		s.clientLock.Unlock()
		v.signal.SendLinkInfo(link)
		if err != nil {
			lib.Println("send error:", err, link.Id)
			s.DelClient(clientId)
			return
		}
		if v.tunnel == nil {
			err = errors.New("tunnel获取错误")
			return
		} else {
			tunnel = v.tunnel
		}
		v.Lock()
		v.linkMap[link.Id] = link
		v.Unlock()
		if !s.waitStatus(clientId, link.Id) {
			err = errors.New("连接失败")
			return
		}
	} else {
		s.clientLock.Unlock()
		err = errors.New("客户端未连接")
	}
	return
}

//得到一个tcp隧道
func (s *Bridge) GetTunnel(id int, en, de int, crypt, mux bool) (conn *lib.Conn, err error) {
	s.clientLock.Lock()
	defer s.clientLock.Unlock()
	if v, ok := s.Client[id]; !ok {
		err = errors.New("客户端未连接")
	} else {
		conn = v.tunnel
	}
	return
}

//得到一个通信通道
func (s *Bridge) GetSignal(id int) (conn *lib.Conn, err error) {
	s.clientLock.Lock()
	defer s.clientLock.Unlock()
	if v, ok := s.Client[id]; !ok {
		err = errors.New("客户端未连接")
	} else {
		conn = v.signal
	}
	return
}

//删除通信通道
func (s *Bridge) DelClient(id int) {
	s.closeClient(id)
}

func (s *Bridge) verify(id int) bool {
	for k := range s.RunList {
		if k == id {
			return true
		}
	}
	return false
}
func (s *Bridge) GetStatus(clientId int) {
	s.clientLock.Lock()
	client := s.Client[clientId]
	s.clientLock.Unlock()

	if client == nil {
		return
	}
	for {
		if id, status, err := client.signal.GetConnStatus(); err != nil {
			s.closeClient(clientId)
			return
		} else {
			client.Lock()
			client.linkStatusMap[id] = status
			client.Unlock()
		}
	}
}

func (s *Bridge) clientCopy(clientId int) {

	s.clientLock.Lock()
	client := s.Client[clientId]
	s.clientLock.Unlock()

	for {
		if id, err := client.tunnel.GetLen(); err != nil {
			s.closeClient(clientId)
			lib.Println("读取msg id 错误", err, id)
			break
		} else {
			client.Lock()
			if link, ok := client.linkMap[id]; ok {
				client.Unlock()
				if content, err := client.tunnel.GetMsgContent(link); err != nil {
					lib.PutBufPoolCopy(content)
					s.closeClient(clientId)
					lib.Println("read msg content error", err, "close client")
					break
				} else {
					if len(content) == len(lib.IO_EOF) && string(content) == lib.IO_EOF {
						if link.Conn != nil {
							link.Conn.Close()
						}
					} else {
						if link.UdpListener != nil && link.UdpRemoteAddr != nil {
							link.UdpListener.WriteToUDP(content, link.UdpRemoteAddr)
						} else {
							link.Conn.Write(content)
						}
						link.Flow.Add(0, len(content))
					}
					lib.PutBufPoolCopy(content)
				}
			} else {
				client.Unlock()
				continue
			}
		}
	}

}
