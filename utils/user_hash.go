package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strconv"
)

// HashUserID hashes the user ID using SHA-256 with a salt (JWT_SECRET) and returns the hexadecimal representation.
func HashUserID(userID uint) string {
	hasher := sha256.New()

	// Retrieve the JWT_SECRET from environment variables
	saltBytes, _ := hex.DecodeString(os.Getenv("JWT_SECRET"))
	salt := string(saltBytes)

	// Combine the userID and salt
	data := strconv.FormatUint(uint64(userID), 10) + salt

	// Write the combined data to the hasher
	hasher.Write([]byte(data))

	return hex.EncodeToString(hasher.Sum(nil))
}
