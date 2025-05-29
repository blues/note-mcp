package main

type NotehubCredentials struct {
	Username string
	Password string
	Token    string
}

// func GetNotehubCredentials() (NotehubCredentials, error) {
// 	envFile, err := godotenv.Read(".env")
// 	if err != nil {
// 		return NotehubCredentials{}, err
// 	}

// 	 notecard.

// 	envFileUsername := envFile["NOTEHUB_USERNAME"]
// 	envFilePassword := envFile["NOTEHUB_PASSWORD"]

// 	return NotehubCredentials{
// 		Username: envFileUsername,
// 		Password: envFilePassword,
// 	}, nil
// }
