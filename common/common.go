package common

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

// GetRegionofIP finds the region of IP addrress using ipapi.co's API
func GetRegionofIP(ipaddr string) (string, error) {
	if strings.Contains(ipaddr, "[::1]") {
		return "Uttarakhand", nil
	}

	resp, err := http.Get(fmt.Sprintf("https://ipapi.co/%s/region/", ipaddr))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return "", err
	}

	return string(b), err
}

func GetIPAddr(r *http.Request) string {

	headerIP := r.Header.Get("X-Forwarded-For")
	if headerIP == "" {
		return r.RemoteAddr
	} else {
		return headerIP
	}
}
