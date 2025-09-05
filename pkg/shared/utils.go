package shared

import (
	"fmt"
	"net"
)

func ExtractIP(conn *net.TCPConn) (string, error) {
  addr, ok := conn.RemoteAddr().(*net.TCPAddr)
  if !ok {
    return "", fmt.Errorf("remote address is not a TCP address")
  }
  return addr.IP.String(), nil
}
