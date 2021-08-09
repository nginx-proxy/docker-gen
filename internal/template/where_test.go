package template

import (
	"testing"

	"github.com/nginx-proxy/docker-gen/internal/context"
)

func TestWhere(t *testing.T) {
	containers := []*context.RuntimeContainer{
		{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost",
			},
			ID: "1",
			Addresses: []context.Address{
				{
					IP:    "172.16.42.1",
					Port:  "80",
					Proto: "tcp",
				},
			},
		},
		{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo2.localhost",
			},
			ID: "2",
			Addresses: []context.Address{
				{
					IP:    "172.16.42.1",
					Port:  "9999",
					Proto: "tcp",
				},
			},
		},
		{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo3.localhost",
			},
			ID: "3",
		},
		{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo2.localhost",
			},
			ID: "4",
		},
	}

	tests := templateTestList{
		{`{{where . "Env.VIRTUAL_HOST" "demo1.localhost" | len}}`, containers, `1`},
		{`{{where . "Env.VIRTUAL_HOST" "demo2.localhost" | len}}`, containers, `2`},
		{`{{where . "Env.VIRTUAL_HOST" "demo3.localhost" | len}}`, containers, `1`},
		{`{{where . "Env.NOEXIST" "demo3.localhost" | len}}`, containers, `0`},
		{`{{where .Addresses "Port" "80" | len}}`, containers[0], `1`},
		{`{{where .Addresses "Port" "80" | len}}`, containers[1], `0`},
		{
			`{{where . "Value" 5 | len}}`,
			[]struct {
				Value int
			}{
				{Value: 5},
				{Value: 3},
				{Value: 5},
			},
			`2`,
		},
	}

	tests.run(t, "where")
}

func TestWhereNot(t *testing.T) {
	containers := []*context.RuntimeContainer{
		{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost",
			},
			ID: "1",
			Addresses: []context.Address{
				{
					IP:    "172.16.42.1",
					Port:  "80",
					Proto: "tcp",
				},
			},
		},
		{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo2.localhost",
			},
			ID: "2",
			Addresses: []context.Address{
				{
					IP:    "172.16.42.1",
					Port:  "9999",
					Proto: "tcp",
				},
			},
		},
		{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo3.localhost",
			},
			ID: "3",
		},
		{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo2.localhost",
			},
			ID: "4",
		},
	}

	tests := templateTestList{
		{`{{whereNot . "Env.VIRTUAL_HOST" "demo1.localhost" | len}}`, containers, `3`},
		{`{{whereNot . "Env.VIRTUAL_HOST" "demo2.localhost" | len}}`, containers, `2`},
		{`{{whereNot . "Env.VIRTUAL_HOST" "demo3.localhost" | len}}`, containers, `3`},
		{`{{whereNot . "Env.NOEXIST" "demo3.localhost" | len}}`, containers, `4`},
		{`{{whereNot .Addresses "Port" "80" | len}}`, containers[0], `0`},
		{`{{whereNot .Addresses "Port" "80" | len}}`, containers[1], `1`},
		{
			`{{whereNot . "Value" 5 | len}}`,
			[]struct {
				Value int
			}{
				{Value: 5},
				{Value: 3},
				{Value: 5},
			},
			`1`,
		},
	}

	tests.run(t, "whereNot")
}

func TestWhereExist(t *testing.T) {
	containers := []*context.RuntimeContainer{
		{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost",
				"VIRTUAL_PATH": "/api",
			},
			ID: "1",
		},
		{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo2.localhost",
			},
			ID: "2",
		},
		{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo3.localhost",
				"VIRTUAL_PATH": "/api",
			},
			ID: "3",
		},
		{
			Env: map[string]string{
				"VIRTUAL_PROTO": "https",
			},
			ID: "4",
		},
	}

	tests := templateTestList{
		{`{{whereExist . "Env.VIRTUAL_HOST" | len}}`, containers, `3`},
		{`{{whereExist . "Env.VIRTUAL_PATH" | len}}`, containers, `2`},
		{`{{whereExist . "Env.NOT_A_KEY" | len}}`, containers, `0`},
		{`{{whereExist . "Env.VIRTUAL_PROTO" | len}}`, containers, `1`},
	}

	tests.run(t, "whereExist")
}

func TestWhereNotExist(t *testing.T) {
	containers := []*context.RuntimeContainer{
		{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost",
				"VIRTUAL_PATH": "/api",
			},
			ID: "1",
		},
		{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo2.localhost",
			},
			ID: "2",
		},
		{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo3.localhost",
				"VIRTUAL_PATH": "/api",
			},
			ID: "3",
		},
		{
			Env: map[string]string{
				"VIRTUAL_PROTO": "https",
			},
			ID: "4",
		},
	}

	tests := templateTestList{
		{`{{whereNotExist . "Env.VIRTUAL_HOST" | len}}`, containers, `1`},
		{`{{whereNotExist . "Env.VIRTUAL_PATH" | len}}`, containers, `2`},
		{`{{whereNotExist . "Env.NOT_A_KEY" | len}}`, containers, `4`},
		{`{{whereNotExist . "Env.VIRTUAL_PROTO" | len}}`, containers, `3`},
	}

	tests.run(t, "whereNotExist")
}

func TestWhereSomeMatch(t *testing.T) {
	containers := []*context.RuntimeContainer{
		{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost",
			},
			ID: "1",
		},
		{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo2.localhost,demo4.localhost",
			},
			ID: "2",
		},
		{
			Env: map[string]string{
				"VIRTUAL_HOST": "bar,demo3.localhost,foo",
			},
			ID: "3",
		},
		{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo2.localhost",
			},
			ID: "4",
		},
	}

	tests := templateTestList{
		{`{{whereAny . "Env.VIRTUAL_HOST" "," (split "demo1.localhost" ",") | len}}`, containers, `1`},
		{`{{whereAny . "Env.VIRTUAL_HOST" "," (split "demo2.localhost,lala" ",") | len}}`, containers, `2`},
		{`{{whereAny . "Env.VIRTUAL_HOST" "," (split "something,demo3.localhost" ",") | len}}`, containers, `1`},
		{`{{whereAny . "Env.NOEXIST" "," (split "demo3.localhost" ",") | len}}`, containers, `0`},
	}

	tests.run(t, "whereAny")
}

func TestWhereRequires(t *testing.T) {
	containers := []*context.RuntimeContainer{
		{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost",
			},
			ID: "1",
		},
		{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo2.localhost,demo4.localhost",
			},
			ID: "2",
		},
		{
			Env: map[string]string{
				"VIRTUAL_HOST": "bar,demo3.localhost,foo",
			},
			ID: "3",
		},
		{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo2.localhost",
			},
			ID: "4",
		},
	}

	tests := templateTestList{
		{`{{whereAll . "Env.VIRTUAL_HOST" "," (split "demo1.localhost" ",") | len}}`, containers, `1`},
		{`{{whereAll . "Env.VIRTUAL_HOST" "," (split "demo2.localhost,lala" ",") | len}}`, containers, `0`},
		{`{{whereAll . "Env.VIRTUAL_HOST" "," (split "demo3.localhost" ",") | len}}`, containers, `1`},
		{`{{whereAll . "Env.NOEXIST" "," (split "demo3.localhost" ",") | len}}`, containers, `0`},
	}

	tests.run(t, "whereAll")
}

func TestWhereLabelExists(t *testing.T) {
	containers := []*context.RuntimeContainer{
		{
			Labels: map[string]string{
				"com.example.foo": "foo",
				"com.example.bar": "bar",
			},
			ID: "1",
		},
		{
			Labels: map[string]string{
				"com.example.bar": "bar",
			},
			ID: "2",
		},
	}

	tests := templateTestList{
		{`{{whereLabelExists . "com.example.foo" | len}}`, containers, `1`},
		{`{{whereLabelExists . "com.example.bar" | len}}`, containers, `2`},
		{`{{whereLabelExists . "com.example.baz" | len}}`, containers, `0`},
	}

	tests.run(t, "whereLabelExists")
}

func TestWhereLabelDoesNotExist(t *testing.T) {
	containers := []*context.RuntimeContainer{
		{
			Labels: map[string]string{
				"com.example.foo": "foo",
				"com.example.bar": "bar",
			},
			ID: "1",
		},
		{
			Labels: map[string]string{
				"com.example.bar": "bar",
			},
			ID: "2",
		},
	}

	tests := templateTestList{
		{`{{whereLabelDoesNotExist . "com.example.foo" | len}}`, containers, `1`},
		{`{{whereLabelDoesNotExist . "com.example.bar" | len}}`, containers, `0`},
		{`{{whereLabelDoesNotExist . "com.example.baz" | len}}`, containers, `2`},
	}

	tests.run(t, "whereLabelDoesNotExist")
}

func TestWhereLabelValueMatches(t *testing.T) {
	containers := []*context.RuntimeContainer{
		{
			Labels: map[string]string{
				"com.example.foo": "foo",
				"com.example.bar": "bar",
			},
			ID: "1",
		},
		{
			Labels: map[string]string{
				"com.example.bar": "BAR",
			},
			ID: "2",
		},
	}

	tests := templateTestList{
		{`{{whereLabelValueMatches . "com.example.foo" "^foo$" | len}}`, containers, `1`},
		{`{{whereLabelValueMatches . "com.example.foo" "\\d+" | len}}`, containers, `0`},
		{`{{whereLabelValueMatches . "com.example.bar" "^bar$" | len}}`, containers, `1`},
		{`{{whereLabelValueMatches . "com.example.bar" "^(?i)bar$" | len}}`, containers, `2`},
		{`{{whereLabelValueMatches . "com.example.bar" ".*" | len}}`, containers, `2`},
		{`{{whereLabelValueMatches . "com.example.baz" ".*" | len}}`, containers, `0`},
	}

	tests.run(t, "whereLabelValueMatches")
}
