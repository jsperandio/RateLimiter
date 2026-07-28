package configs

import "github.com/joho/godotenv"

func LoadEnv(path string) {
	if path == "" {
		return
	}

	_ = godotenv.Load(path)
}
