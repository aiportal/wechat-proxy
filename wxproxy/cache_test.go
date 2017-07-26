package wxproxy

import (
	"testing"
	"time"
	"fmt"
)

func TestCacheMap(t *testing.T) {
	ts_data := []struct{
		key string
		value interface{}
	}{
		{
			key: "aaa",
			value: struct{
				a string
				b string
			}{a: "a", b: "b"},
		},
		{
			key: "bbb",
			value: "test",
		},
	}

	var cache = NewCacheMap(1 * time.Second, 1)
	for _, v := range ts_data {
		cache.Set(v.key, v.value)
	}

	for _, v := range ts_data {
		value, ok := cache.Get(v.key)
		if !ok {
			t.Fatal()
		}
		if value != v.value {
			t.Fatal()
		}
	}

	time.Sleep(2 * time.Second)
	cache.Clean()
	fmt.Println(cache.Len())
	if cache.Len() > 1 {
		t.Fatal()
	}
}