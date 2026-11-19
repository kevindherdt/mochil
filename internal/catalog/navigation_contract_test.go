package catalog

import (
	"encoding/json"
	"os"
	"testing"
)

func TestNavigationBucketContract(t *testing.T) {
	data, err := os.ReadFile("../../contracts/navigation_buckets.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Title  string `json:"title"`
		Bucket string `json:"bucket"`
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) == 0 {
		t.Fatal("empty navigation contract")
	}
	for _, test := range cases {
		if got := Bucket(test.Title); got != test.Bucket {
			t.Errorf("%q: got %q want %q", test.Title, got, test.Bucket)
		}
	}
}
