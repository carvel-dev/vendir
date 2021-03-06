package download

import (
	"errors"
	"fmt"
	"net/http"
)

type Range struct {
	Lower      int64
	Upper      int64
	HTTPHeader http.Header
}

func NewRange(lower int64, upper int64, httpHeader http.Header) (Range){
	return Range {
		 lower,
		 upper,
		httpHeader,
	}
}

type Ranger struct {
	numHunks int
}

func NewRanger(hunks int) Ranger {
	return Ranger{
		numHunks: hunks,
	}
}

func (r Ranger) BuildRange(contentLength int64) ([]Range, error) {
	var ranges []Range

	if contentLength == 0 {
		return ranges, errors.New("content length cannot be zero")
	}

	hunkSize := contentLength / int64(r.numHunks)
	if hunkSize == 0 {
		hunkSize = 2
	}

	iterations := (contentLength / hunkSize)
	remainder := contentLength % int64(hunkSize)

	for i := int64(0); i < int64(iterations); i++ {
		lowerByte := i * hunkSize
		upperByte := ((i + 1) * hunkSize) - 1
		if i == int64(iterations-1) {
			upperByte += remainder
		}
		formattedBytes := fmt.Sprintf("bytes=%d-%d", lowerByte, upperByte)
		ranges = append(ranges, Range{
			Lower:      lowerByte,
			Upper:      upperByte,
			HTTPHeader: http.Header{"Range": []string{formattedBytes}},
		})
	}

	return ranges, nil
}
