package admin

import (
	"strconv"
	"strings"
)

// parseRustdeskReply translates hbbs and hbbr console replies into structured values.
func parseRustdeskReply(target, command, option, output string) map[string]interface{} {
	output = strings.TrimSpace(strings.ReplaceAll(output, "\r\n", "\n"))
	result := map[string]interface{}{
		"kind": "notice", "summary": "服务器已接收操作请求",
		"target": target, "command": command,
	}
	lines := make([]string, 0)
	for _, line := range strings.Split(output, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}
	list := func(rows []map[string]string, empty string) {
		result["kind"] = "list"
		result["rows"] = rows
		result["count"] = len(rows)
		if len(rows) == 0 {
			result["summary"] = empty
		} else {
			result["summary"] = "共 " + strconv.Itoa(len(rows)) + " 条记录"
		}
	}
	write := option != "" &&
		(command == "relay-servers" || command == "always-use-relay" ||
			command == "blacklist-add" || command == "blacklist-remove" ||
			command == "blocklist-add" || command == "blocklist-remove" ||
			command == "downgrade-threshold" || command == "downgrade-start-check" ||
			command == "limit-speed" || command == "total-bandwidth" ||
			command == "single-bandwidth" ||
			((command == "ip-blocker" || command == "ip-changes") &&
				strings.HasSuffix(strings.TrimSpace(option), " -")))
	if write {
		result["summary"] = "操作请求已发送至服务器，请重新查询确认当前状态"
		return result
	}
	switch command {
	case "always-use-relay":
		value := strings.TrimSpace(strings.TrimPrefix(output, "ALWAYS_USE_RELAY:"))
		if strings.EqualFold(value, "true") || strings.EqualFold(value, "false") {
			result["kind"] = "boolean"
			result["active"] = strings.EqualFold(value, "true")
			if result["active"] == true {
				result["summary"] = "当前要求通过中继服务器建立连接"
			} else {
				result["summary"] = "当前允许优先尝试直接连接"
			}
			return result
		}
	case "relay-servers":
		entries := make([]map[string]string, 0, len(lines))
		for _, line := range lines {
			entries = append(entries, map[string]string{"address": line})
		}
		list(entries, "当前没有单独配置中继服务器地址")
		return result
	case "blacklist", "blocklist":
		if option != "" && len(lines) == 1 &&
			(strings.EqualFold(lines[0], "true") || strings.EqualFold(lines[0], "false")) {
			result["kind"] = "boolean"
			result["active"] = strings.EqualFold(lines[0], "true")
			if result["active"] == true {
				result["summary"] = "该 IP 地址在列表中"
			} else {
				result["summary"] = "该 IP 地址不在列表中"
			}
			return result
		}
		entries := make([]map[string]string, 0, len(lines))
		for _, line := range lines {
			entries = append(entries, map[string]string{"address": line})
		}
		list(entries, "当前列表中没有 IP 地址")
		return result
	case "ip-blocker", "ip-changes":
		entries := make([]map[string]string, 0)
		if len(lines) > 0 {
			if total, err := strconv.Atoi(lines[0]); err == nil && total >= 0 {
				result["total"] = total
				lines = lines[1:]
			}
		}
		for _, line := range lines {
			name, detail, ok := strings.Cut(line, ": ")
			if !ok {
				name, detail = "", line
			}
			entries = append(entries, map[string]string{"name": name, "detail": detail})
		}
		list(entries, "当前没有符合条件的记录")
		result["kind"] = "table"
		if total, ok := result["total"]; ok {
			result["summary"] = "服务器记录总数：" + strconv.Itoa(total.(int))
		}
		return result
	case "downgrade-threshold", "downgrade-start-check",
		"limit-speed", "total-bandwidth", "single-bandwidth":
		if len(lines) == 1 {
			raw := lines[0]
			unit := ""
			switch {
			case strings.HasSuffix(raw, "Mb/s"):
				unit = "Mb/s"
			case strings.HasSuffix(raw, "s"):
				unit = "秒"
			}
			number := strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(raw, "Mb/s"), "s"))
			if value, err := strconv.ParseFloat(number, 64); err == nil {
				result["kind"] = "metric"
				result["value"] = value
				result["unit"] = unit
				result["summary"] = "服务器当前设置"
				return result
			}
		}
	case "usage":
		entries := make([]map[string]string, 0, len(lines))
		for _, line := range lines {
			address, values, ok := strings.Cut(line, ": ")
			parts := strings.Fields(values)
			if ok && len(parts) == 5 &&
				strings.HasSuffix(parts[0], "s") && strings.HasSuffix(parts[1], "MB") {
				entries = append(entries, map[string]string{
					"address": address,
					"seconds": strings.TrimSuffix(parts[0], "s"),
					"megabytes": strings.TrimSuffix(parts[1], "MB"),
					"peak_kbps": strings.TrimSuffix(parts[2], "kb/s"),
					"average_kbps": strings.TrimSuffix(parts[3], "kb/s"),
					"current_kbps": strings.TrimSuffix(parts[4], "kb/s"),
				})
			} else {
				entries = append(entries, map[string]string{"detail": line})
			}
		}
		list(entries, "当前没有可显示的中继使用记录")
		result["kind"] = "usage"
		return result
	case "h":
		names := map[string]string{
			"relay-servers": "中继服务器地址", "reload-geo": "更新地区数据库",
			"ip-blocker": "IP 封禁记录", "ip-changes": "设备 IP 变化",
			"always-use-relay": "强制使用中继", "test-geo": "地区识别测试",
			"must-login": "要求登录", "blacklist-add": "添加限速名单",
			"blacklist-remove": "移除限速名单", "blacklist": "限速名单",
			"blocklist-add": "添加阻止名单", "blocklist-remove": "移除阻止名单",
			"blocklist": "阻止名单", "downgrade-threshold": "连接速度降级阈值",
			"downgrade-start-check": "降级检查间隔", "limit-speed": "速度上限",
			"total-bandwidth": "中继总带宽", "single-bandwidth": "单连接带宽",
			"usage": "中继传输统计",
		}
		entries := make([]map[string]string, 0, len(lines))
		for _, line := range lines {
			name := strings.TrimSpace(strings.SplitN(line, "(", 2)[0])
			label := names[name]
			if label == "" {
				label = name
			}
			entries = append(entries, map[string]string{"name": label, "command": name})
		}
		list(entries, "服务器未返回功能清单")
		return result
	case "test-geo":
		if output != "" {
			result["kind"] = "value"
			result["value"] = strings.Trim(output, "\"")
			result["summary"] = "服务器选择的中继地址"
			return result
		}
	}
	if len(lines) != 0 {
		entries := make([]map[string]string, 0, len(lines))
		for _, line := range lines {
			entries = append(entries, map[string]string{"detail": line})
		}
		list(entries, "当前没有可显示的信息")
	} else {
		result["kind"] = "empty"
		result["summary"] = "当前没有可显示的信息"
	}
	return result
}
