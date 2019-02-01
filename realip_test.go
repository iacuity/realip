package realip

import (
	"net/http"
	"testing"
)

func TestIsPrivateAddr(t *testing.T) {
	testData := map[string]bool{
		"127.0.0.0":   true,
		"10.0.0.0":    true,
		"169.254.0.0": true,
		"192.168.0.0": true,
		"::1":         true,
		"fc00::":      true,

		"172.15.0.0": false,
		"172.16.0.0": true,
		"172.31.0.0": true,
		"172.32.0.0": false,

		"147.12.56.11": false,
	}

	for addr, isLocal := range testData {
		isPrivate, err := isPrivateAddress(addr)
		if err != nil {
			t.Errorf("fail processing %s: %v", addr, err)
		}

		if isPrivate != isLocal {
			format := "%s should "
			if !isLocal {
				format += "not "
			}
			format += "be local address"

			t.Errorf(format, addr)
		}
	}
}

func TestRealIP(t *testing.T) {
	// Create type and function for testing
	type testIP struct {
		name     string
		request  *http.Request
		expected string
	}

	newRequest := func(remoteAddr, xRealIP string, xForwardedFor ...string) *http.Request {
		h := http.Header{}
		h.Set("X-Real-IP", xRealIP)
		for _, address := range xForwardedFor {
			h.Set("X-Forwarded-For", address)
		}

		return &http.Request{
			RemoteAddr: remoteAddr,
			Header:     h,
		}
	}

	// Create test data
	publicAddr1 := "144.12.54.87"
	publicAddr2 := "119.14.55.11"
	localAddr := "127.0.0.0"

	testData := []testIP{
		{
			name:     "No header",
			request:  newRequest(publicAddr1, ""),
			expected: publicAddr1,
		}, {
			name:     "Has X-Forwarded-For",
			request:  newRequest("", "", publicAddr1),
			expected: publicAddr1,
		}, {
			name:     "Has multiple X-Forwarded-For",
			request:  newRequest("", "", localAddr, publicAddr1, publicAddr2),
			expected: publicAddr2,
		}, {
			name:     "Has X-Real-IP",
			request:  newRequest("", publicAddr1),
			expected: publicAddr1,
		},
	}

	// Run test
	for _, v := range testData {
		if actual := FromRequest(v.request); v.expected != actual {
			t.Errorf("%s: expected %s but get %s", v.name, v.expected, actual)
		}
	}
}

func TestClientIPFromRequest(t *testing.T) {
	// Create type and function for testing
	type testIP struct {
		name     string
		request  *http.Request
		expected string
	}

	newRequest := func(remoteAddr string, xRealIP string, xForwardedFor string) *http.Request {
		headerList := http.Header{}
		headerList.Set("X-Real-IP", xRealIP)
		headerList.Set("X-Forwarded-For", xForwardedFor)

		return &http.Request{
			RemoteAddr: remoteAddr,
			Header:     headerList,
		}
	}

	// Create test data
	privateAddr := "127.0.0.1:1234"
	publicAddr := "144.12.54.87"
	multiAddresses := " 127.0.0.1 , 115.98.247.136, 144.12.54.87:1234 , 119.14.55.11:1234 ,"
	multi_publicAddr := "115.98.247.136"

	testData := []testIP{
		{
			name:     "Has remoteAddr",
			request:  newRequest(publicAddr, "", privateAddr),
			expected: publicAddr,
		}, {
			name:     "Has X-Real-IP",
			request:  newRequest(privateAddr, publicAddr, ""),
			expected: publicAddr,
		}, {
			name:     "Has X-Forwarded-For",
			request:  newRequest(privateAddr, privateAddr, publicAddr),
			expected: publicAddr,
		}, {
			name:     "Has multiple X-Forwarded-For",
			request:  newRequest(privateAddr, privateAddr, multiAddresses),
			expected: multi_publicAddr,
		}, {
			name:     "Has all private IP",
			request:  newRequest(privateAddr, privateAddr, privateAddr),
			expected: "",
		}, {
			name:     "Has no IP at all",
			request:  newRequest("--", ":", ":"),
			expected: "",
		},
	}

	// Run test
	for _, v := range testData {
		if actual, source := ClientIPFromRequest(v.request); v.expected != actual {
			t.Errorf("Case:%s, expected:[%s], actual:[%s] from source:%s", v.name, v.expected, actual, source)
			break
		}
	}
}
