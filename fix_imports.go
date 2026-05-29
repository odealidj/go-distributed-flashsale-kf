package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"os/exec"
)

func main() {
	services := []string{"api-gateway", "product-service", "inventory-service", "order-service", "payment-service"}

	for _, s := range services {
		path := filepath.Join(s, "cmd", s, "main.go")
		content, err := os.ReadFile(path)
		if err != nil {
			fmt.Println("Error reading:", path, err)
			continue
		}
		
		strContent := string(content)
		// Fix missing context
		if !strings.Contains(strContent, "\"context\"") {
			strContent = strings.Replace(strContent, "import (", "import (\n\t\"context\"", 1)
		}
		// Remove unused trace
		if !strings.Contains(strContent, "trace.") && strings.Contains(strContent, "\"go.opentelemetry.io/otel/trace\"") {
			strContent = strings.Replace(strContent, "\t\"go.opentelemetry.io/otel/trace\"\n", "", 1)
		}
		if !strings.Contains(strContent, "otel.") && strings.Contains(strContent, "\"go.opentelemetry.io/otel\"") {
			strContent = strings.Replace(strContent, "\t\"go.opentelemetry.io/otel\"\n", "", 1)
		}

		err = os.WriteFile(path, []byte(strContent), 0644)
		if err != nil {
			fmt.Println("Error writing:", path, err)
		}

		// goimports to be safe
		exec.Command("go", "fmt", path).Run()
	}
}
