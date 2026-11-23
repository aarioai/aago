package httpsvr

import (
	json "encoding/json/v2"
	"fmt"
	"testing"

	"github.com/aarioai/aago/aa/ae"
	"github.com/aarioai/aago/aa/atype"
)

func jsons(v any, e *ae.Error) string {
	if e != nil {
		fmt.Println(e.String())
		return ""
	}
	s, err := json.Marshal(v, json.Deterministic(true))
	if err != nil {
		return ""
	}
	return string(s)
}

func TestStringifyPayloadFields(t *testing.T) {
	type Child struct {
		Test string `json:"test"`
	}
	type y struct {
		Child
		Bad    atype.ImagePaths `json:"bad"`
		Tmp    atype.ImgSrc     `json:"-"`
		t      []atype.ImgSrc   `json:"images"`
		Y      *int             `json:"y"`
		Img    *atype.ImgSrc    `json:"img"`
		Images []atype.ImgSrc   `json:"ims"`
	}
	type x struct {
		Child
		A string       `json:"a"`
		B int64        `json:"b"`
		C int          `json:"c"`
		Y *y           `json:"y"`
		M *atype.Money `json:"m"`
		N atype.Money  `json:"money"`
	}
	img := atype.ImgSrc{}
	ims := []atype.ImgSrc{img}
	yy := 10000
	y0 := y{Y: &yy, Images: ims}
	a := x{A: "LOVE", B: 100, C: 300, Y: &y0}
	as := `{"a":"LOVE","b":"100","c":300,"m":null,"money":"0","test":"","y":{"bad":{},"img":null,"ims":[{"allowed":null,"alter_url_pattern":"","base_url":"","checksum":"","crop_url_pattern":"","filesize":0,"filetype":"","height":0,"info":"","jsonkey":"","path":"","provider":0,"url":"","url_pattern":"","width":0}],"test":"","y":10000}}`
	s := jsons(StringifyBigintFields(a, "json"))
	if s != as {
		t.Errorf("%s --> %s", s, as)
	}
	b := []x{a}
	bs := "[" + as + "]"
	s = jsons(StringifyBigintFields(b, "json"))
	if s != bs {
		t.Errorf("%s --> %s", s, bs)
	}
}
