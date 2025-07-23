package sandbox

import (
	"errors"
	"fmt"
	"log"
	"os"
	"runtime"
	"time"
)

const CodeStorageFolder = "/sandbox/code"

func WriteToTempFile(b []byte) (string, error) {
	// using nano second to avoid filename collision in highly concurrent requests
	id := fmt.Sprintf("%v", time.Now().UnixNano())
	err := os.WriteFile(shellFilename(id), b, 0777)

	if err != nil && errors.Is(err, os.ErrNotExist) {
		// may be the folder absent. so trying to create it
		err = os.Mkdir(CodeStorageFolder, os.ModePerm)
		if err != nil {
			log.Println("failed creating folder:", CodeStorageFolder)
			return id, err
		}
		log.Println("created folder:", CodeStorageFolder)
		// second attempt
		err = os.WriteFile(shellFilename(id), b, 0777)
	}

	return id, err
}

func shellFilename(timestamp string) string {
	return fmt.Sprintf("%v/%v.sh", CodeStorageFolder, timestamp)
}

func LogWithLocation(msg string) {
	// skip=1 表示上一層呼叫者
	if pc, file, line, ok := runtime.Caller(1); ok {
		fn := runtime.FuncForPC(pc)
		fmt.Printf("[LOG] %s:%d [%s] → %s\n", file, line, fn.Name(), msg)
	} else {
		fmt.Println("[LOG] unknown location →", msg)
	}
}
