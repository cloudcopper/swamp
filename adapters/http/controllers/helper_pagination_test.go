package controllers

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHelperPagination(t *testing.T) {
	assert := assert.New(t)

	fill := func(n int) []string {
		ret := []string{}
		for x := 0; x < n; x++ {
			ret = append(ret, fmt.Sprintf("text-%v", x))
		}
		return ret
	}
	testCases := []struct {
		desc    string
		inData  []string
		inPage  int
		perPage int
		outData []string
		outPage int
	}{
		{"40 vs 20", fill(40), 1, 20, fill(20), 1},
		{"empty input", nil, 1, 20, nil, 1},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			r := &http.Request{
				URL: &url.URL{
					RawQuery: fmt.Sprintf("page=%v", tC.inPage),
				},
			}
			outData, outPage := helperPagination(r, tC.inData, tC.perPage)
			assert.Equal(tC.outData, outData)
			assert.Equal(tC.outPage, outPage)
		})
	}
}
