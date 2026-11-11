package admin

import (
	"errors"
	"math"
	"net"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

var serverCommandDeviceID = regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,128}$`)
var serverCommandHost = regexp.MustCompile(`^[A-Za-z0-9._:\[\]-]+$`)

// validateRustdeskCommand accepts the documented RustDesk server operations and checks each argument.
func validateRustdeskCommand(target, command, option string) error {
	if !utf8.ValidString(option) || len(option) > 1800 || strings.ContainsAny(option, "\r\n\x00;|&") {
		return errors.New("参数包含不受支持的字符或超出长度限制")
	}
	option = strings.TrimSpace(option)
	all := model.SysRelayServerCmds
	if target == model.ServerCmdTargetIdServer {
		all = model.SysIdServerCmds
	} else if target != model.ServerCmdTargetRelayServer {
		return errors.New("服务器类型无效")
	}
	found := false
	for _, predefined := range all {
		if predefined.Cmd == command {
			found = true
			break
		}
	}
	if !found {
		return errors.New("该服务器不支持所选管理操作")
	}
	parts := strings.Fields(option)
	requireIP := func(input string) error {
		if net.ParseIP(input) == nil {
			return errors.New("请填写有效的 IP 地址")
		}
		return nil
	}
	requireNumber := func(input string) error {
		number, err := strconv.ParseFloat(input, 64)
		if err != nil || math.IsNaN(number) || math.IsInf(number, 0) || number < 0 || number > 1000000000 {
			return errors.New("数值应介于 0 与 1000000000 之间")
		}
		return nil
	}
	noArguments := func() error {
		if option != "" { return errors.New("该操作不需要参数") }
		return nil
	}
	switch command {
	case "h", "usage":
		return noArguments()
	case "relay-servers":
		if option == "" { return nil }
		servers := strings.Split(option, ",")
		if len(servers) > 32 { return errors.New("最多可填写 32 个服务器地址") }
		for _, server := range servers {
			server = strings.TrimSpace(server)
			if server == "" || len(server) > 255 || !serverCommandHost.MatchString(server) {
				return errors.New("中继服务器地址格式无效")
			}
		}
	case "ip-blocker":
		if option == "" { return nil }
		if len(parts) < 1 || len(parts) > 2 || len(parts) == 2 && parts[1] != "-" {
			return errors.New("请输入 IP 地址或封禁编号，可在末尾填写 - 解除封禁")
		}
		if net.ParseIP(parts[0]) == nil {
			if id, err := strconv.ParseUint(parts[0], 10, 64); err != nil || id == 0 {
				return errors.New("请填写有效的 IP 地址或封禁编号")
			}
		}
	case "ip-changes":
		if option == "" { return nil }
		if len(parts) < 1 || len(parts) > 2 || len(parts) == 2 && parts[1] != "-" ||
			!serverCommandDeviceID.MatchString(parts[0]) {
			return errors.New("请输入有效的设备 ID，可在末尾填写 - 清除记录")
		}
	case "always-use-relay":
		if option != "" && option != "y" && option != "n" {
			return errors.New("此设置只接受 y 或 n")
		}
	case "test-geo":
		if len(parts) != 2 { return errors.New("需要填写两个 IP 地址") }
		for _, item := range parts { if err := requireIP(item); err != nil { return err } }
	case "blacklist", "blacklist-add", "blacklist-remove",
		"blocklist", "blocklist-add", "blocklist-remove":
		if len(parts) != 1 { return errors.New("此操作需要一个 IP 地址") }
		return requireIP(parts[0])
	case "downgrade-threshold", "downgrade-start-check", "limit-speed",
		"total-bandwidth", "single-bandwidth":
		if option == "" { return nil }
		if len(parts) != 1 { return errors.New("请输入一个数值") }
		return requireNumber(parts[0])
	default:
		return errors.New("不支持的管理操作")
	}
	return nil
}
