package controllers

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHelperPagination(t *testing.T) {
	assert := require.New(t)

	data := func() []string {
		ret, max := []string{}, 100
		for x := 0; x < max; x++ {
			ret = append(ret, fmt.Sprintf("text-%v", x))
		}
		return ret
	}()
	testCases := []struct {
		desc    string
		inPage  int
		perPage int
		inData  []string
		outPage int
		outData []string
	}{
		{"empty input", 1, 20, nil, 1, nil},
		{"page 1(20) from [40]", 1, 20, data[0:40], 1, data[0:20]},
		{"page 0(20) from [40]", 0, 20, data[0:40], 1, data[0:20]},
		{"page -1(20) from [40]", -1, 20, data[0:40], 1, data[0:20]},
		{"page 2(20) from [38]", 2, 20, data[0:38], 2, data[20:38]},
		{"page 3(20) from [38]", 3, 20, data[0:38], 2, data[20:38]},
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
