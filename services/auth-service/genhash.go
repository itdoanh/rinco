package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"golang.org/x/crypto/argon2"
)

func main() {
	salt := make([]byte, 16)
	rand.Read(salt)
	key := argon2.IDKey([]byte("rinco_dev_password"), salt, 3, 64*1024, 2, 32)
	fmt.Printf("$argon2id$v=19$m=65536,t=3,p=2$%s$%s\n",
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key))
}
