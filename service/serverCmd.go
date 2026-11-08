package service

import (
	"bytes"
	"io"
	"fmt"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"net"
	"time"
)

type ServerCmdService struct{}

// List
func (is *ServerCmdService) List(page, pageSize uint) (res *model.ServerCmdList) {
	res = &model.ServerCmdList{}
	res.Page = int64(page)
	res.PageSize = int64(pageSize)
	tx := DB.Model(&model.ServerCmd{})
	tx.Count(&res.Total)
	tx.Scopes(Paginate(page, pageSize))
	tx.Find(&res.ServerCmds)
	return
}

// Info
func (is *ServerCmdService) Info(id uint) *model.ServerCmd {
	u := &model.ServerCmd{}
	DB.Where("id = ?", id).First(u)
	return u
}

// Delete
func (is *ServerCmdService) Delete(u *model.ServerCmd) error {
	return DB.Delete(u).Error
}

// Create
func (is *ServerCmdService) Create(u *model.ServerCmd) error {
	res := DB.Create(u).Error
	return res
}

// SendCmd 发送命令
func (is *ServerCmdService) SendCmd(port int, cmd string, arg string) (string, error) {
	//组装命令
	cmd = cmd + " " + arg
	res, err := is.SendSocketCmd("v6", port, cmd)
	if err == nil {
		return res, nil
	}
	//v6连接失败，尝试v4
	res, err = is.SendSocketCmd("v4", port, cmd)
	if err == nil {
		return res, nil
	}
	return "", err
}

// SendSocketCmd
func (is *ServerCmdService) SendSocketCmd(ty string, port int, cmd string) (string, error) {
	addr, network := "127.0.0.1", "tcp4"
	if ty == "v6" {
		addr, network = "::1", "tcp6"
	}
	conn, err := net.DialTimeout(network, net.JoinHostPort(addr, fmt.Sprint(port)), 3*time.Second)
	if err != nil { return "", err }
	defer conn.Close()
	if err := conn.SetWriteDeadline(time.Now().Add(3*time.Second)); err != nil { return "", err }
	if _, err := conn.Write([]byte(cmd)); err != nil { return "", err }
	const maxReply = 65536
	var response bytes.Buffer
	buffer := make([]byte, 4096)
	if err := conn.SetReadDeadline(time.Now().Add(2*time.Second)); err != nil { return "", err }
	for response.Len() < maxReply {
		remaining := maxReply - response.Len()
		if remaining < len(buffer) { buffer = buffer[:remaining] }
		n, readErr := conn.Read(buffer)
		if n > 0 {
			response.Write(buffer[:n])
			conn.SetReadDeadline(time.Now().Add(200*time.Millisecond))
		}
		if readErr != nil {
			if netErr, ok := readErr.(net.Error); ok && netErr.Timeout() && response.Len() > 0 {
				break
			}
			if readErr == io.EOF { break }
			return "", readErr
		}
	}
	return response.String(), nil
}

func (is *ServerCmdService) Update(f *model.ServerCmd) error {
	return DB.Model(f).Updates(f).Error
}
