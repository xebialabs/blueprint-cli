package util

import (
	"testing"
)

func TestMd5HashFromFilteredMap(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]interface{}
		filter  []interface{}
		invert  bool
		want    string
		wantErr bool
	}{
		{
			"create hash from given map based on filters",
			map[string]interface{}{
				"foo":  "bar",
				"bar":  10,
				"fooo": true,
				"baar": "hello",
			},
			[]interface{}{"foo", "bar"},
			false,
			"d2639145905a67631756942d27ce0f1f",
			false,
		},
		{
			"create the same hash from given map based on filters when other values change",
			map[string]interface{}{
				"foo":  "bar",
				"bar":  10,
				"fooo": true,
				"baar": "hellozz",
			},
			[]interface{}{"foo", "bar"},
			false,
			"d2639145905a67631756942d27ce0f1f",
			false,
		},
		{
			"create hash from given map based on filters with inverse",
			map[string]interface{}{
				"foo":  "bar",
				"bar":  10,
				"fooo": true,
				"baar": "hello",
			},
			[]interface{}{"fooo", "baar"},
			true,
			"d2639145905a67631756942d27ce0f1f",
			false,
		},
		{
			"create the same hash from given map based on filters when other values change with inverse",
			map[string]interface{}{
				"foo":  "bar",
				"bar":  10,
				"fooo": true,
				"baar": "hellozz",
			},
			[]interface{}{"fooo", "baar"},
			true,
			"d2639145905a67631756942d27ce0f1f",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Md5HashFromFilteredMap(tt.params, tt.filter, tt.invert)
			if (err != nil) != tt.wantErr {
				t.Errorf("Md5HashFromMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Md5HashFromMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMd5HashFromMap(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]interface{}
		want    string
		wantErr bool
	}{
		{
			"return error when marshal fails",
			map[string]interface{}{
				"foo": make(chan int),
			},
			"",
			true,
		},
		{
			"create hash from given map",
			map[string]interface{}{
				"foo":  "bar",
				"bar":  10,
				"fooo": true,
				"baar": "hello",
			},
			"8e95292b8f637910ed4af0c262177e02",
			false,
		},
		{
			"should create same hash from given map with different order as above",
			map[string]interface{}{
				"fooo": true,
				"baar": "hello",
				"bar":  10,
				"foo":  "bar",
			},
			"8e95292b8f637910ed4af0c262177e02",
			false,
		},
		{
			"should create differenr hash for different value",
			map[string]interface{}{
				"fooz": false,
				"baar": "hello",
				"bar":  10,
				"foo":  "bar",
			},
			"08c930ed345772acdd63d6790d94f7c8",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Md5HashFromMap(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("Md5HashFromMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Md5HashFromMap() = %v, want %v", got, tt.want)
			}
		})
	}
}
