package pelicantun

import "fmt"

type addr struct {
	Ip     string
	Port   int
	IpPort string // Ip:Port
}

func NewAddr2(ip string, port int) addr {
	return addr{
		Ip:     ip,
		Port:   port,
		IpPort: fmt.Sprintf("%s:%d", ip, port),
	}
}

func NewAddr1(ipport string) addr {
	var ip string
	var port int

	n, err := fmt.Scanf("%s:%d", &ip, &port)
	if n != 2 || err != nil {
		panic(fmt.Sprintf("bad scanf of ipport input '%s': got %d scanned and err: '%s'", ipport, n, err))
	}
	return addr{
		Ip:     ip,
		Port:   port,
		IpPort: ipport,
	}
}
