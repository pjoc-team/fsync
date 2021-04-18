package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"flag"
	oss2 "github.com/pjoc-team/fsync/pkg/storage/backend/oss"
	"io/ioutil"
	"math/rand"
)

var (
	endpoint  string
	bucket    string
	secretID  string
	secretKey string
)

func init() {
	flag.StringVar(&endpoint, "endpoint", "https://cos.ap-guangzhou.myqcloud.com", "endpoint")
	flag.StringVar(&bucket, "bucket", "backup-1251070767", "bucket")
	flag.StringVar(&secretID, "secretID", "[changeSecretID]", "secretID")
	flag.StringVar(&secretKey, "secretKey", "[changeSecretKey]", "secretKey")
}

func main() {
	conf := &oss2.Conf{
		Endpoint:  endpoint,
		Bucket:    bucket,
		SecretID:  secretID,
		SecretKey: secretKey,
	}
	blockSize := 3 * 1024 * 1024
	s, err := oss2.NewOssStorage(
		conf,
		blockSize,
		true,
	)
	if err != nil {
		panic(err.Error())
	}

	ctx := context.Background()
	writer, err := s.Create(ctx, "a/b/test.txt")
	if err != nil {
		panic(err.Error())
	}
	defer func() {
		err2 := writer.Close()
		if err2 != nil {
			panic(err2.Error())
		}
	}()

	var expect bytes.Buffer

	for i := 0; i < 3; i++ {
		b := make([]byte, blockSize)
		if _, err := rand.Read(b); err != nil {
			panic(err.Error())
		}
		_, err = writer.Write(b)
		expect.Write(b)
		if err != nil {
			panic(err.Error())
		}
	}

	reader, err := s.Get(ctx, "a/b/test.txt")
	if err != nil {
		panic(err.Error())
	}

	all, err := ioutil.ReadAll(reader)
	if err != nil {
		panic(err.Error())
	}
	err2 := reader.Close()
	if err2 != nil {
		panic(err2.Error())
	}
	// fmt.Println("=========get file=========:", base64.RawStdEncoding.EncodeToString(all))
	if base64.RawStdEncoding.EncodeToString(all) != base64.RawStdEncoding.EncodeToString(
		expect.Bytes(),
	) {
		panic("not expect")
	}
}
