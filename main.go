package main

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	// shutdown functions
	shutdownFunctions := make([]func(context.Context), 0)

	// signal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(interrupt)

	// errgroup
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		r := gin.Default()
		server := &http.Server{Addr: ":8080", Handler: r}

		shutdownFunctions = append(shutdownFunctions, func(ctx context.Context) {
			err := server.Close()
			if err != nil {
				fmt.Println("failed to close http server!")
			} else {
				fmt.Println("succeed to close http server!")
			}
		})

		r.GET("/ping", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "pong",
			})
		}).
			GET("/", func(c *gin.Context) {
				_, err := c.Writer.Write([]byte("hello!\n"))
				if err != nil {
					fmt.Printf("error: %v\n", err.Error())
				}
				//c.String(http.StatusOK, "", "hello!\n")
			})
		err := server.ListenAndServe()
		return err
	})

	//http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
	//	_, err := writer.Write([]byte("hello!\n"))
	//	if err != nil {
	//		fmt.Printf("write error: %v \n", err.Error())
	//	}
	//})
	//err := http.ListenAndServe(":8080", nil)
	select {
	case <-ctx.Done():
		break
	case <-interrupt:
		break
	}

	timeout, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	for _, shutdown := range shutdownFunctions {
		shutdown(timeout)
	}
	err := g.Wait()
	if err != nil {
		panic(err)
	}

}
