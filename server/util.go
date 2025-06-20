package main

import (
    "crypto/rand"
    "log"
)

func generateShortCode(length int) string {
    const chars = "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
    b := make([]byte, length)
    _, err := rand.Read(b)
    if err != nil {
        log.Fatal("Error generating random string:", err)
    }
    for i := 0; i < length; i++ {
        b[i] = chars[int(b[i])%len(chars)]
    }
    return string(b)
}