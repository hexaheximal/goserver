package compression

import (
	"compress/gzip"
	"io/ioutil"
	"bytes"
)

// TODO: Compression levels?

func CompressData(source []byte) []byte {
	var buf bytes.Buffer
	
	zw := gzip.NewWriter(&buf)

	_, err := zw.Write(source)
	
	if err != nil {
		panic(err)
	}

	if err := zw.Close(); err != nil {
		panic(err)
	}

	return buf.Bytes()
}

func DecompressData(source []byte) []byte {
	reader := bytes.NewReader(source)
	
	gzreader, err := gzip.NewReader(reader)
	
	if err != nil {
		panic(err)
	}
	
	output, err2 := ioutil.ReadAll(gzreader)
	
	if err2 != nil {
		panic(err2)
	}
	
	return output
}