package yaml

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"regexp"
)

const sep = "---"

type Yaml struct {
	Source string
	Values map[interface{}]interface{}
}

var sepr, _ = regexp.Compile(`\n---\n`)

func Group(ys []Yaml, key interface{}) map[interface{}][]Yaml {
	res := make(map[interface{}][]Yaml)

	for _, y := range ys {
		v := y.Values[key]
		res[v] = append(res[v], y)
	}

	return res
}

func Parse(s string) ([]Yaml, error) {
	ss := sepr.Split(s, -1)
	ys := make([]Yaml, 0)

	for _, v := range ss {
		y, err := parse(v)

		if err != nil {
			return []Yaml{}, err
		}

		if len(y.Values) > 0 {
			ys = append(ys, *y)
		}
	}

	return ys, nil
}

func parse(s string) (*Yaml, error) {
	v := make(map[interface{}]interface{})
	err := yaml.Unmarshal([]byte(s), &v)

	if err != nil {
		return &Yaml{}, err
	}

	y := Yaml{
		Source: s,
		Values: v,
	}

	return &y, err
}

func ParseFile(f *os.File) ([]Yaml, error) {
	bs, err := ioutil.ReadAll(f)

	if err != nil {
		return []Yaml{}, err
	}

	return Parse(string(bs))
}

func String(ys []Yaml) (string, error) {
	s := ""

	for i, y := range ys {
		res, err := yaml.Marshal(y.Values)

		if err != nil {
			return "", err
		}

		s += fmt.Sprintf("%s", string(res))

		if i < len(ys)-1 {
			s += fmt.Sprintf("%s\n", sep)
		}
	}

	return s, nil
}
