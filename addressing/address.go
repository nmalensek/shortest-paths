package addressing

import (
	"log"
	"net"
)

//GetIP creates a UDP connection and retrieves the IP address the caller is using for outbound communication.
func GetIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	local := conn.LocalAddr().(*net.UDPAddr)

	return local.IP
}
