package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestLink(t *testing.T) {
	dst1, _ := url.Parse("http://dst/one")
	dst2, _ := url.Parse("http://dst/two?a=1")

	l := links{
		"link1": Target{dst1},
		"link2": Target{dst2},
	}

	for _, test := range []struct {
		req  *http.Request
		want string
	}{
		{
			httptest.NewRequest("GET", "/link1", nil),
			"http://dst/one",
		},
		{
			httptest.NewRequest("GET", "/link1?x=42", nil),
			"http://dst/one?x=42",
		},
		{
			httptest.NewRequest("GET", "/link1/suffix", nil),
			"http://dst/one/suffix",
		},
		{
			httptest.NewRequest("GET", "/link1/suffix?x=42", nil),
			"http://dst/one/suffix?x=42",
		},
		{
			httptest.NewRequest("GET", "/link2", nil),
			"http://dst/two?a=1",
		},
		{
			httptest.NewRequest("GET", "/link2?x=42", nil),
			"http://dst/two?a=1&x=42",
		},
		{
			httptest.NewRequest("GET", "/link2?a=4", nil),
			"http://dst/two?a=4",
		},
		{
			httptest.NewRequest("GET", "/link2?a=4&x=42", nil),
			"http://dst/two?a=4&x=42",
		},
	} {
		w := httptest.NewRecorder()
		l.ServeHTTP(w, test.req)
		got, err := w.Result().Location()
		if err != nil {
			t.Error(err)
		}
		if got.String() != test.want {
			t.Errorf("FAIL:\t%s\n\tgot:  %s\n\twant: %s",
				test.req.URL.String(), got, test.want)
		} else {
			t.Logf("SUCCESS:\t%s\t->\t%s",
				test.req.URL.String(), test.want)
		}
	}

}
