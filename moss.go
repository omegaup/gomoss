// Gomoss is Go interface for Moss client http://theory.stanford.edu/~aiken/moss/
package gomoss

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/url"
)

// MossRequest contains Moss client ID and configuration parameter
type MossRequest struct {
	// ID is the Moss server user ID
	ID uint32
	// Language is programing language type
	Language string
	// Ignore limit
	IgnoreLimit uint16
	// Comment is comment that will appear on result
	Comment string
	// EndPoint is Moss plagiarism detector endpoint
	EndPoint *Address
	// ExperimentalServer is flag utilize experimental server
	ExperimentalServer uint8
	// MatchingNumber is number of matching files to show in the result
	MatchingNumber uint16
	// DirectoryMode is a mode to add files with directory
	DirectoryMode uint8
	// Code is source codes that will be sent to Moss server
	Code codeList
	// BaseCode is base codes that will be sent to Moss server
	BaseCode codeList
}

// Address is end point of Moss plagiarism detector
type Address struct {
	Host string
	Port string
}

// code struct
type code struct {
	// reader is opened file reader
	reader io.Reader
	// displayName is custom file name that will be shown in Moss server
	displayName string
}

type codeList struct {
	list []*code
}

var (
	ErrLanguageNotSupported = errors.New("Language is not supported")
)

// NewRequest creates a new instance of MossRequest
func NewRequest(userID uint32) *MossRequest {
	return &MossRequest{
		ID:                 userID,
		Language:           "c",
		IgnoreLimit:        10,
		Comment:            "",
		EndPoint:           &Address{"moss.stanford.edu", "7690"},
		ExperimentalServer: 0,
		MatchingNumber:     250,
		DirectoryMode:      0,
	}
}

// Add adds Code into Moss struct
func (c *codeList) Add(reader io.Reader, displayName string) {
	c.list = append(c.list, &code{reader, displayName})
}

// uploadFile uploads a code to Moss server
func uploadFile(conn net.Conn, code *code, lang string, index int) error {
	buff, err := ioutil.ReadAll(code.reader)
	if err != nil {
		return err
	}
	header := fmt.Sprintf("file %d %s %d %s\n", index, lang, len(buff), code.displayName)
	conn.Write([]byte(header))
	conn.Write(buff)
	return nil
}

// Submit function sends codes and base codes with its configuration to Moss server
func (request *MossRequest) Submit(ctx context.Context) (*url.URL, error) {
	conn, err := net.Dial("tcp", net.JoinHostPort(request.EndPoint.Host, request.EndPoint.Port))
	defer conn.Close()

	if err != nil {
		return nil, err
	}

	conn.Write([]byte(fmt.Sprintf("moss %d\n", request.ID)))
	conn.Write([]byte(fmt.Sprintf("directory %d\n", request.DirectoryMode)))
	conn.Write([]byte(fmt.Sprintf("X %d\n", request.ExperimentalServer)))
	conn.Write([]byte(fmt.Sprintf("maxmatches %d\n", request.IgnoreLimit)))
	conn.Write([]byte(fmt.Sprintf("show %d\n", request.MatchingNumber)))
	conn.Write([]byte(fmt.Sprintf("language %s\n", request.Language)))

	buff := make([]byte, 1024)
	_, err = conn.Read(buff)

	if err != nil {
		return nil, err
	}
	if bytes.Equal(buff, []byte("yes")) {
		fmt.Println(string(buff))
		return nil, ErrLanguageNotSupported
	}

	for _, code := range request.BaseCode.list {
		select {
		case <-ctx.Done():
			return nil, nil
		default:
			uploadFile(conn, code, request.Language, 0)
		}
	}

	for index, code := range request.Code.list {
		select {
		case <-ctx.Done():
			return nil, nil
		default:
			uploadFile(conn, code, request.Language, index+1)
		}
	}
	conn.Write([]byte(fmt.Sprintf("query 0 %s\n", request.Comment)))

	scanner := bufio.NewScanner(conn)
	scanner.Scan()
	urlText := scanner.Text()

	conn.Write([]byte("end\n"))

	urlStruct, err := url.Parse(urlText)
	if err != nil {
		return nil, err
	}
	return urlStruct, nil
}
