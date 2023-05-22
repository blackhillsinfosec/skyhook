package config

import (
	"errors"
	"fmt"
	"net"
)

// FindInterface iterates over all network interfaces and
// attempts to find one that matches either the interface's
// name or IP address.
func FindInterface(target string) (string, error) {

	var err error
	var ifaces []net.Interface
	if ifaces, err = net.Interfaces(); err != nil {
		return target, err
	}

	var addrs []net.Addr
	var ipFound bool

	for _, iface := range ifaces {

		if addrs, err = iface.Addrs(); err != nil {
			return target, err
		}

		if len(addrs) < 1 && iface.Name == target {
			return target, errors.New("failed to get address from interface name")
		}

		if iface.Name == target {

			//====================================
			// PULL IP ADDRESS FROM INTERFACE NAME
			//====================================

			target = addrs[0].(*net.IPNet).IP.String()
			ipFound = true
			break

		} else {

			//===============================
			// SEARCH FOR MATCHING IP ADDRESS
			//===============================

			for _, iA := range addrs {

				if target == iA.(*net.IPNet).IP.String() {
					ipFound = true
					break
				}

			}
		}

		if ipFound {
			break
		}
	}

	if !ipFound {
		return target, errors.New(fmt.Sprintf(
			"failed to find requested bind interface %v", target))
	}

	return target, err
}
