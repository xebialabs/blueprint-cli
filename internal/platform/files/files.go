package files

import (
	"os"
)

func Close(fs []*os.File) []error {
	res := make([]error, 0)

	for _, f := range fs {
		err := f.Close()

		if err != nil {
			res = append(res, err)
		}
	}

	return res
}

func Open(fs ...string) ([]*os.File, error) {
	res := make([]*os.File, 0)

	for _, fn := range fs {
		src, err := os.Open(fn)

		if err != nil {
			return res, err
		}

		res = append(res, src)
	}

	return res, nil
}
