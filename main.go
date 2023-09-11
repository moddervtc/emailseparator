package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	maxGoroutines   = 10 // Número máximo de goroutines em paralelo
	emailsProcessed = make(map[string]bool)
	mutex           = sync.Mutex{}
)

func processLine(line, selectedProvider string, writer *bufio.Writer) {
	// Processar a linha da mesma forma que antes
	partes := strings.FieldsFunc(strings.TrimSpace(line), func(r rune) bool {
		return r == ':' || r == '|'
	})
	if len(partes) >= 2 {
		email := strings.TrimSpace(partes[0])
		if strings.Contains(email, "@") {
			provedor := strings.Split(email, "@")[1]
			if strings.Contains(provedor, selectedProvider) {
				// Verificar se o email já foi processado
				mutex.Lock()
				defer mutex.Unlock()
				if !emailsProcessed[email] {
					fmt.Fprintf(writer, "%s\n", line)
					emailsProcessed[email] = true
				}
			}
		}
	}
}

func main() {
	fmt.Print("Enter the provider you want to separate (ex: hotmail): ")
	var selectedProvider string
	fmt.Scanln(&selectedProvider)

	if _, err := os.Stat("RESULTS"); os.IsNotExist(err) {
		os.Mkdir("RESULTS", os.ModeDir)
	}

	arquivo, err := os.Open("emails.txt")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer arquivo.Close()

	scanner := bufio.NewScanner(arquivo)

	// Criar um canal para controlar o número de goroutines em execução
	semaphore := make(chan struct{}, maxGoroutines)

	totalEmails := 0 // Total de emails no arquivo
	emailsProcessedCount := 0

	fmt.Println("Starting the process...")

	// Abrir arquivo de saída com um buffer maior (64 KB)
	nomeArquivo := fmt.Sprintf("RESULTS/%s.txt", selectedProvider)
	arquivoProvedor, err := os.OpenFile(nomeArquivo, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer arquivoProvedor.Close()

	writer := bufio.NewWriterSize(arquivoProvedor, 1024*1024) // 1 MB buffer
	defer writer.Flush()

	// Calcular o número total de emails
	for scanner.Scan() {
		totalEmails++
	}

	// Retornar ao início do arquivo
	arquivo.Seek(0, 0)
	scanner = bufio.NewScanner(arquivo)

	// Calcular a hora estimada de término
	startTime := time.Now()
	estimatedTime := time.Duration(totalEmails) * 100 * time.Millisecond

	for scanner.Scan() {
		line := scanner.Text()
		semaphore <- struct{}{} // Adquirir um slot do semáforo

		go func(line string) {
			defer func() {
				<-semaphore // Liberar o slot do semáforo após a conclusão
			}()

			processLine(line, selectedProvider, writer)
			emailsProcessedCount++

			// Exibir o progresso
			fmt.Printf("\rProgress: %d/%d (%.2f%%)", emailsProcessedCount, totalEmails, float64(emailsProcessedCount)/float64(totalEmails)*100)

			// Exibir o tempo estimado de término
			elapsedTime := time.Since(startTime)
			remainingTime := estimatedTime - elapsedTime
			fmt.Printf(" Estimated time remaining: %s", remainingTime.Round(time.Second))
		}(line)
	}

	// Aguardar a conclusão de todas as goroutines
	for i := 0; i < maxGoroutines; i++ {
		semaphore <- struct{}{}
	}

	fmt.Println("\nProcess concluded.")
}
