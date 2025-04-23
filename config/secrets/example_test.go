package secrets_test

import (
	"fmt"
	"log"
	"os"

	"github.com/lone-faerie/mqttop/config/secrets"
)

func Example() {
	// Setup secret file for testing
	err := os.MkdirAll("/run/secrets", 0660)
	if err != nil {
		log.Fatal(err)
	}
	err = os.WriteFile("/run/secrets/foo", []byte("Hello, world!"), 0660)
	if err != nil {
		log.Fatal(err)
	}
	// Delete file after example
	defer os.Remove("/run/secrets/foo")

	// Get secret
	s := "!secret foo"
	s, ok := secrets.CutPrefix(s)
	if !ok {
		log.Fatal(s, "is not a secret")
	}
	secret, err := secrets.Read(s)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(secret)
}
