package gomoss

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"
	"testing"
)

func TestMossInit(t *testing.T) {
	mossID := uint32(12345678)
	request := NewRequest(mossID)

	switch {
	case request.ID != mossID:
		t.Fail()
	case request.IgnoreLimit != 10:
		t.Fail()
	case request.ExperimentalServer != 0:
		t.Fail()
	case request.MatchingNumber != 250:
		t.Fail()
	case request.DirectoryMode != 0:
		t.Fail()
	case request.Language != "c":
		t.Fail()
	}
}

func TestAddCode(t *testing.T) {
	request := NewRequest(uint32(12345678))

	aFilepath := "testdata/helloworld.c"
	aFile, err := os.Open(aFilepath)
	if err != nil {
		t.Fatalf("Unable to open source code (%s)", aFilepath)
	}
	defer aFile.Close()
	request.Code.Add(aFile, "helloworld.c")

	var mossBytes []byte
	request.Code.list[0].reader.Read(mossBytes)
	var aBytes []byte
	aFile.Read(aBytes)

	if !bytes.Equal(mossBytes, aBytes) {
		t.Errorf("Local code and moss code reader are not equal (%s)", aFilepath)
	}
}

func TestAddBaseCode(t *testing.T) {
	request := NewRequest(uint32(12345678))

	aFilepath := "testdata/helloworld.c"
	aFile, err := os.Open(aFilepath)
	if err != nil {
		t.Fatalf("Unable to open source code (%s)", aFilepath)
	}
	request.BaseCode.Add(aFile, "ppp")

	var mossBytes []byte
	request.BaseCode.list[0].reader.Read(mossBytes)
	var aBytes []byte
	aFile.Read(aBytes)

	if !bytes.Equal(mossBytes, aBytes) {
		t.Errorf("Local base code and moss base code reader are not equal (%s)", aFilepath)
	}
}

func ExampleNewRequest() {
	// Get this ID from https://theory.stanford.edu/~aiken/moss/
	// ID must be uint32
	mossID := uint32(12345678)

	// Create new request from Moss ID
	request := NewRequest(mossID)

	// Change the default parameters
	// For more information about parameters see http://moss.stanford.edu/general/scripts/mossnet
	request.IgnoreLimit = 30
	request.MatchingNumber = 100
	request.Language = "cc"
}

func ExampleMossRequest_Submit() {
	mossID := uint32(12345678)
	request := NewRequest(mossID)

	sampleFile, err := os.Open("testdata/helloworld.c")
	if err != nil {
		panic(err)
	}
	request.Code.Add(sampleFile, "helloworld.c")

	// or load files with for iteration
	for _, filename := range []string{"a.c", "b.c"} {
		f, err := os.Open(path.Join("testdata", filename))
		if err != nil {
			panic(err)
		}
		defer f.Close()
		request.Code.Add(f, filename)
	}

	// Add BaseCode
	// If BaseCode is supplied program code that appears in the BaseCode is not counted in matches
	baseFile, _ := os.Open("testdata/base.c")
	request.BaseCode.Add(baseFile, "base.c")
	defer sampleFile.Close()

	ctx := context.Background()
	url, err := request.Submit(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Println(url)
}
