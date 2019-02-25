package blueprint

import (
	"reflect"
	"testing"
)

func Test_processCustomExpression(t *testing.T) {
	type args struct {
		exStr      string
		parameters map[string]interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			"should fail when using undefined parameter",
			args{
				"FooUndefined > 100",
				map[string]interface{}{
					"foo": "foo",
				},
			},
			nil,
			true,
		},
		{
			"should return true when parameter is evaluated",
			args{
				"Foo > 10",
				map[string]interface{}{
					"Foo": 100,
				},
			},
			true,
			false,
		},
		{
			"should return true when parameter is evaluated",
			args{
				"Foo && Bar",
				map[string]interface{}{
					"Foo": true,
					"Bar": false,
				},
			},
			false,
			false,
		},
		{
			"should return false when expression is evaluated",
			args{
				"Foo < 10",
				map[string]interface{}{
					"Foo": 100,
				},
			},
			false,
			false,
		},
		{
			"should return Bar when ternary expression is evaluated",
			args{
				"Foo > 10 ? Foo : Bar",
				map[string]interface{}{
					"Foo": 10,
					"Bar": 200,
				},
			},
			float64(200),
			false,
		},
		{
			"should return an array when ternary expression is evaluated",
			args{
				"Foo ? Bar : (1, 2, 3)",
				map[string]interface{}{
					"Foo": true,
					"Bar": []string{"test", "foo"},
				},
			},
			[]string{"test", "foo"},
			false,
		},
		{
			"should return true when logical expression is evaluated",
			args{
				"Foo == 10 && Bar != 10",
				map[string]interface{}{
					"Foo": 10,
					"Bar": 200,
				},
			},
			true,
			false,
		},
		{
			"should return '100' when expression is evaluated",
			args{
				"Foo + Bar",
				map[string]interface{}{
					"Foo": 75,
					"Bar": 25,
				},
			},
			float64(100),
			false,
		},
		{
			"should return 'foo+bar' when expression is evaluated",
			args{
				"Foo + '+' + Bar",
				map[string]interface{}{
					"Foo": "foo",
					"Bar": "bar",
				},
			},
			"foo+bar",
			false,
		},
		{
			"should return length of Foo when expression is evaluated",
			args{
				"strlen(Foo)",
				map[string]interface{}{
					"Foo": "foo0",
				},
			},
			float64(4),
			false,
		},
		{
			"should return max of 2 numbers when expression is evaluated",
			args{
				"max(2, 1)",
				map[string]interface{}{},
			},
			float64(2),
			false,
		},
		{
			"should return rounded value of a number when expression is evaluated",
			args{
				"round(2.12556)",
				map[string]interface{}{},
			},
			float64(2),
			false,
		},
		{
			"should return true when a complex logical expression is evaluated",
			args{
				"((Foo == 10 && Bar != 10) ? Bar: Foo) == 200 && (Fooz == 'test' || 'test' == Fooz) && (Fooz + Foo == 'test10') && Foo != 20",
				map[string]interface{}{
					"Foo":  10,
					"Bar":  200,
					"Fooz": "test",
				},
			},
			true,
			false,
		},
		{
			"should return 3 when a complex math expression is evaluated",
			args{
				"ceil(min(Foo / Bar * Fooz, Foo * 0.5 ) * round(2.8956))",
				map[string]interface{}{
					"Foo":  100,
					"Bar":  200,
					"Fooz": 1.88888,
				},
			},
			float64(3),
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ProcessCustomExpression(tt.args.exStr, tt.args.parameters)
			if (err != nil) != tt.wantErr {
				t.Errorf("processCustomExpression() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("processCustomExpression() = %v, want %v", got, tt.want)
			}
		})
	}
}
