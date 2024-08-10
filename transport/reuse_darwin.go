//go:build darwin

package transport

import "syscall"

func SetReuseOpt(network, address string, c syscall.RawConn) error {
	return nil
}
