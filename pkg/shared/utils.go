package shared

import "net"

func ExtractIp(conn *net.TCPConn) string {
  addr := conn.RemoteAddr().(*net.TCPAddr)
  return addr.IP.String()
}
