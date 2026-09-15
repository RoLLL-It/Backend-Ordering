package main

func Main(args map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"statusCode": 200,
		"body":       `{"status":"ok"}`,
		"headers":    map[string]interface{}{"Content-Type": "application/json"},
	}
}

func main() {}
