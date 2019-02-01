package realip

import (
	"errors"
	"net"
	"net/http"
	"strings"
)

var cidrs []*net.IPNet

func init() {
	maxCidrBlocks := []string{
		"127.0.0.1/8",    // localhost
		"10.0.0.0/8",     // 24-bit block
		"172.16.0.0/12",  // 20-bit block
		"192.168.0.0/16", // 16-bit block
		"169.254.0.0/16", // link local address
		"::1/128",        // localhost IPv6
		"fc00::/7",       // unique local address IPv6
		"fe80::/10",      // link local address IPv6
	}

	cidrs = make([]*net.IPNet, len(maxCidrBlocks))
	for i, maxCidrBlock := range maxCidrBlocks {
		_, cidr, _ := net.ParseCIDR(maxCidrBlock)
		cidrs[i] = cidr
	}
}

// isLocalAddress works by checking if the address is under private CIDR blocks.
// List of private CIDR blocks can be seen on :
//
// https://en.wikipedia.org/wiki/Private_network
//
// https://en.wikipedia.org/wiki/Link-local_address
func isPrivateAddress(address string) (bool, error) {
	ipAddress := net.ParseIP(address)
	if ipAddress == nil {
		return false, errors.New("address is not valid")
	}

	for i := range cidrs {
		if cidrs[i].Contains(ipAddress) {
			return true, nil
		}
	}

	return false, nil
}

func getIPfromHostPort(hostPort string) string {
	var err error
	remoteIP := hostPort
	// If there are colon in remote address, remove the port number
	// otherwise, return remote address as is
	if strings.ContainsRune(hostPort, ':') {
		remoteIP, _, err = net.SplitHostPort(hostPort)
		if err != nil {
			remoteIP = ""
		}
	}
	remoteIP = strings.TrimSpace(remoteIP)
	return remoteIP
}

func isValidPublicIP(ip string) bool {
	isPrivate, err := isPrivateAddress(ip)
	return (err == nil && !isPrivate)
}

// ClientIPFromRequest return client's real public IP address from http request headers.
func ClientIPFromRequest(r *http.Request) (ip string, source string) {
	clientIP := ""

	//TODO:
	//Check Standard headers
	//Forwarded: for=192.0.2.60:1234;proto=http;by=203.0.113.43

	//Try from X-Forwarded-For and return the first global address
	xForwardedFor := r.Header.Get("X-Forwarded-For")
	for _, address := range strings.Split(xForwardedFor, ",") {
		clientIP = getIPfromHostPort(address)
		if isValidPublicIP(clientIP) {
			return clientIP, "X-Forwarded-For"
		}
	}

	//Try from X-Real-Ip
	clientIP = getIPfromHostPort(r.Header.Get("X-Real-Ip"))
	if isValidPublicIP(clientIP) {
		return clientIP, "X-Real-Ip"
	}

	//Try from X-Client-IP
	clientIP = getIPfromHostPort(r.Header.Get("X-Client-Ip"))
	if isValidPublicIP(clientIP) {
		return clientIP, "X-Client-Ip"
	}

	//Final
	//Try from r.RemoteAddr
	clientIP = getIPfromHostPort(r.RemoteAddr)
	if isValidPublicIP(clientIP) {
		return clientIP, "remoteAddr"
	}

	return "", ""
}

// FromRequest return client's real public IP address from http request headers.
func FromRequest(r *http.Request) string {
	// Fetch header value
	xRealIP := r.Header.Get("X-Real-Ip")
	xForwardedFor := r.Header.Get("X-Forwarded-For")

	// If both empty, return IP from remote address
	if xRealIP == "" && xForwardedFor == "" {
		var remoteIP string

		// If there are colon in remote address, remove the port number
		// otherwise, return remote address as is
		if strings.ContainsRune(r.RemoteAddr, ':') {
			remoteIP, _, _ = net.SplitHostPort(r.RemoteAddr)
		} else {
			remoteIP = r.RemoteAddr
		}

		return remoteIP
	}

	// Check list of IP in X-Forwarded-For and return the first global address
	for _, address := range strings.Split(xForwardedFor, ",") {
		address = strings.TrimSpace(address)
		isPrivate, err := isPrivateAddress(address)
		if !isPrivate && err == nil {
			return address
		}
	}

	// If nothing succeed, return X-Real-IP
	return xRealIP
}

// RealIP is depreciated, use FromRequest instead
func RealIP(r *http.Request) string {
	return FromRequest(r)
}
