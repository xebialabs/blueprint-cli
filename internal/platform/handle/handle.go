package handle

import (
	"fmt"
	"log"
)

func BasicError(pre string, err error) {
	if err != nil {
		if pre != "" {
			panic(fmt.Sprintf("%v: %v", pre, err))
		} else {
			panic(fmt.Sprintf("%v", err))
		}
	}
}

func BasicPanicAsLog() {
	if r := recover(); r != nil {
		log.Println(r)
	}
}
