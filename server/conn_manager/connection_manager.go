package connmanager

import (
	"net"
	"sync"
)

type ConnectionManager struct {
  mu sync.RWMutex
  clientToConn map[string]*net.TCPConn
  connToClient map[*net.TCPConn]string
}

func NewConnectionManager() *ConnectionManager{
  clientToConn := make(map[string]*net.TCPConn)
  connToClient := make(map[*net.TCPConn]string)

  return &ConnectionManager{
    clientToConn: clientToConn,
    connToClient: connToClient,
  }
}

func (cm *ConnectionManager) Register(conn *net.TCPConn, username string) {
  cm.mu.Lock()
  defer cm.mu.Unlock()

  cm.connToClient[conn] = username
  cm.clientToConn[username] = conn
}

func (cm *ConnectionManager) Unregister(conn *net.TCPConn) (string, bool) {
  cm.mu.Lock()
  defer cm.mu.Unlock()

  username, exist := cm.connToClient[conn]
  if exist {
    delete(cm.clientToConn, username)
    delete(cm.connToClient, conn)
  }
  return username, exist
}

func (cm *ConnectionManager) GetUser(conn *net.TCPConn) (string, bool) {
  cm.mu.RLock()
  defer cm.mu.RUnlock()

  username, exist := cm.connToClient[conn]
  return username, exist
}

func (cm *ConnectionManager) GetConn(username string) (*net.TCPConn, bool) {
  cm.mu.RLock()
  defer cm.mu.RUnlock()

  conn, exist := cm.clientToConn[username]
  return conn, exist
}
