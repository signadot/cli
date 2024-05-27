package swaggerWrapper

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/go-openapi/runtime"
	"io"
)

func TextStreamConsumer() runtime.Consumer {
	return runtime.ConsumerFunc(func(reader io.Reader, output interface{}) error {
		if reader == nil {
			return errors.New("TextStreamConsumer requires a reader") // early exit
		}
		if output == nil {
			return errors.New("nil destination for TextStreamConsumer")
		}

		defer func() {
			fmt.Println("ending this")
			fmt.Println(output)
		}()

		for {
			scanner := bufio.NewScanner(reader)
			for scanner.Scan() {
				fmt.Println("L", len(scanner.Text()))
			}
		}

		return nil

	})
}
