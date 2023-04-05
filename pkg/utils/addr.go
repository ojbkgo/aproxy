package utils

import (
	"strconv"
	"strings"
)

func ParseAddr(addr string) (string, []int64) {
	addrs := strings.Split(addr, ":")
	ip := addrs[0]
	ports := strings.Split(addrs[1], ",")
	portInt := make([]int64, len(ports))
	for i := range ports {
		portInt[i], _ = strconv.ParseInt(ports[i], 10, 64)
	}
	return ip, portInt
}
