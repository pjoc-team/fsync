package oss

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"time"
)

func ExampleStorage_CreateUploadURL() {
	ossStorage, err := NewOssStorage(
		&Conf{
			Endpoint:  "https://cos.ap-guangzhou.myqcloud.com",
			Bucket:    "bucketxxx",
			SecretID:  "[secret_id]",
			SecretKey: "[secret_key]",
		}, 1024*1024, true,
	)
	if err != nil {
		panic(err)
	}

	s := (ossStorage).(*storage)
	url, err := s.CreateUploadURL(context.Background(), "test_presigned.txt", 30*time.Minute)
	if err != nil {
		return
	}
	fmt.Println(url)
	bufferString := bytes.NewBufferString("test")
	request, err := newFileUploadRequest(url, map[string]string{}, "test", "test", bufferString)
	if err != nil {
		return
	}
	post, err := http.DefaultClient.Do(request)
	// post, err := http.DefaultClient.Post(url, "image/jpeg", bufferString)
	if err != nil {
		panic(err)
		return
	}
	fmt.Println(post.Status)
	all, err := ioutil.ReadAll(post.Body)
	if err != nil {
		panic(err)
		return
	}
	fmt.Println(string(all))
}

// Creates a new file upload http request with optional extra params
func newFileUploadRequest(
	uri string, params map[string]string, paramName string, name string, reader io.Reader,
) (*http.Request, error) {

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(paramName, name)
	if err != nil {
		return nil, err
	}
	all, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	part.Write(all)

	for key, val := range params {
		_ = writer.WriteField(key, val)
	}
	err = writer.Close()
	if err != nil {
		return nil, err
	}

	return http.NewRequest(http.MethodPut, uri, body)
}
