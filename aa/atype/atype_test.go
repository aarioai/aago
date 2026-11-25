package atype_test

import (
	"net"
	"testing"

	"github.com/aarioai/aago/aa/atype"
)

func TestUnmarshalUint8s(t *testing.T) {
	v, err := atype.UnmarshalUint8s([]byte(`[1, 2,3]`))
	if err != nil {
		t.Fatalf("UnmarshalUint8s: %v", err)
	}
	if len(v) != 3 || v[0] != 1 || v[1] != 2 || v[2] != 3 {
		t.Fatalf("UnmarshalUint8s = %v", v)
	}
	if v, err = atype.UnmarshalUint8s([]byte("null")); err != nil || v != nil {
		t.Fatalf("null => %v, %v", v, err)
	}
	if v, err = atype.UnmarshalUint8s([]byte("[]")); err != nil || len(v) != 0 {
		t.Fatalf("[] => %v, %v", v, err)
	}
	for _, bad := range []string{`[1,x]`, `[1`, `[300]`} {
		if _, err = atype.UnmarshalUint8s([]byte(bad)); err == nil {
			t.Errorf("UnmarshalUint8s(%q) should fail", bad)
		}
	}
}

func TestIPOK(t *testing.T) {
	var ip atype.IP
	if ip.OK() {
		t.Fatal("zero IP should not be OK")
	}

	// An invalid value is not an IP, even when its stored width is 16 bytes.
	var invalid atype.IP
	invalid.String = "0123456789abcdef"
	if invalid.OK() {
		t.Fatal("invalid IP should not be OK")
	}

	if err := ip.Scan([]byte(net.ParseIP("127.0.0.1").To4())); err != nil {
		t.Fatal(err)
	}
	if !ip.OK() || !ip.Is4() || ip.To16() != "127.0.0.1" {
		t.Fatalf("IPv4 IP = %+v", ip)
	}
}

func TestAtype(t *testing.T) {
	zero := atype.New()
	defer zero.Close()
	if zero.String() != "" {
		t.Errorf("String() should return empty string")
	}
	b := 234242342342423.3
	s := zero.Reload(b).String()
	if s != "234242342342423.3" {
		t.Errorf("float64(%f) ==> string(%s)", b, s)
	}
}

func TestAtypeGet(t *testing.T) {
	arr := map[any]any{
		1:      100,
		"name": "Aario",
		"1":    "999",
		"test": map[string]any{
			"nation": "China",
			"city":   "Shenzhen",
		},
		2: map[string]string{
			"sex": "male",
		},
	}

	d := atype.New(arr)
	defer d.Close()
	v, err := d.Get("name")
	if err != nil {
		t.Fatal("get name failed")
	}

	if v.String() != "Aario" {
		t.Errorf(`["name"] %s != Aario`, v.String())
	}

	v, err = d.Get(1)
	if err != nil {
		t.Fatal("get 1 failed")
	}

	i, err := v.Int()
	if i != 100 {
		t.Errorf("[1] %d != 100", i)
	}

	v, err = d.Get("1")
	if err != nil {
		t.Fatal(`get "1" failed`)
	}
	if v.String() != "999" {
		t.Errorf("[\"1\"] %s != 999", v.String())
	}

	v, err = d.Get("test.nation")
	if err != nil {
		t.Fatal("get test.nation failed")
	}
	if v.String() != "China" {
		t.Errorf("test.nation %s != China", v.String())
	}

	v, err = d.Get(2, "sex")
	if err != nil {
		t.Fatal("get 2.sex failed")
	}
	if v.String() != "male" {
		t.Errorf("2.sex %s != male", v.String())
	}
}
