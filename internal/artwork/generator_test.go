package artwork

import (
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
)

func TestGeneratorCoalescesMatchingImageRequests(t *testing.T) {
	root := t.TempDir()
	countPath := filepath.Join(root, "runs")
	commandPath := filepath.Join(root, "extract")
	script := "#!/bin/sh\n" +
		"printf x >> " + strconv.Quote(countPath) + "\n" +
		"sleep 0.05\n" +
		"for last do :; done\n" +
		"printf image > \"$last\"\n"
	if err := os.WriteFile(commandPath, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	generator, err := NewGenerator(filepath.Join(root, "cache"), commandPath, 1<<20, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer generator.Close()
	inputs := make([]*os.File, 2)
	for index := range inputs {
		path := filepath.Join(root, "input-"+strconv.Itoa(index))
		if err := os.WriteFile(path, []byte("source"), 0o600); err != nil {
			t.Fatal(err)
		}
		inputs[index], err = os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
	}
	var wait sync.WaitGroup
	errors := make(chan error, len(inputs))
	for _, input := range inputs {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, imageErr := generator.Image(input, "same-key")
			errors <- imageErr
		}()
	}
	wait.Wait()
	close(errors)
	for imageErr := range errors {
		if imageErr != nil {
			t.Fatal(imageErr)
		}
	}
	runs, err := os.ReadFile(countPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(runs) != "x" {
		t.Fatalf("extractor ran %d times", len(runs))
	}
}
