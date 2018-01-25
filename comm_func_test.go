package comm

import (
	"testing"
)

func TestIpToAton(t *testing.T) {
	val := IpToAton("192.168.0.1")
	if val != 3232235521 {
		t.Error()
	}
}
