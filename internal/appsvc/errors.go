package appsvc

import (
	"errors"
	"strings"
)

func localizeError(err error) error {
	if err == nil {
		return nil
	}
	message := localizeErrorMessage(err.Error())
	if message == err.Error() {
		return err
	}
	return errors.New(message)
}

func localizeErrorMessage(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return message
	}

	lower := strings.ToLower(message)
	prefix := ""
	switch {
	case containsAny(lower, "attempted logon is invalid", "bad username or authentication", "logon failure"):
		prefix = "SMB 账号或密码错误，请检查用户名、密码和域"
	case containsAny(lower, "network name not found", "specified share name cannot be found"):
		prefix = "SMB 共享名不存在，请检查共享名是否与服务器一致"
	case containsAny(lower, "status_access_denied", "access denied", "permission denied"):
		prefix = "权限不足，请检查账号对该目录或文件的访问权限"
	case containsAny(lower, "unknown authority", "certificate signed by unknown authority", "failed to verify certificate"):
		prefix = "TLS 证书不受信任，请导入证书或使用受信任证书"
	case containsAny(lower, "530", "login incorrect"):
		prefix = "FTP 登录失败，请检查用户名和密码"
	case containsAny(lower, "550"):
		prefix = "FTP 权限不足或路径不存在，请检查目录权限和路径"
	case containsAny(lower, "401", "unauthorized"):
		prefix = "WebDAV 认证失败，请检查用户名和密码"
	case containsAny(lower, "404", "not found"):
		prefix = "远端路径不存在，请检查 Base URL、共享名或远端路径"
	case containsAny(lower, "no such host", "connection refused", "i/o timeout", "network is unreachable", "host is down", "cannot connect"):
		prefix = "网络连接失败，请检查主机、端口和网络连通性"
	default:
		return message
	}

	if strings.HasPrefix(message, prefix) {
		return message
	}
	return prefix + "：" + message
}

func containsAny(value string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}
