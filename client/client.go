package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	pb "grpcClientTages/pbs" // Импорт сгенерированного кода

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Подключение к серверу
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewFileServiceClient(conn)

	if len(os.Args) < 2 {
		fmt.Println("Использование:")
		fmt.Println("  go run client.go upload <путь_к_файлу>   - загрузить файл")
		fmt.Println("  go run client.go list                    - показать список файлов")
		fmt.Println("  go run client.go download <имя_файла>    - скачать файл")
		return
	}

	command := os.Args[1]

	switch command {
	case "upload":
		if len(os.Args) < 3 {
			fmt.Println("Укажите путь к файлу для загрузки")
			return
		}
		uploadFile(client, os.Args[2])
	case "list":
		listFiles(client)
	case "download":
		if len(os.Args) < 3 {
			fmt.Println("Укажите имя файла для скачивания")
			return
		}
		downloadFile(client, os.Args[2])
	default:
		fmt.Println("Неизвестная команда:", command)
	}
}

func uploadFile(client pb.FileServiceClient, filePath string) {
	fmt.Printf("Загружаем файл: %s\n", filePath)

	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Не удалось открыть файл: %v", err)
	}
	defer file.Close()

	// Создаем stream для загрузки
	stream, err := client.UploadFile(context.Background())
	if err != nil {
		log.Fatalf("Не удалось создать stream: %v", err)
	}

	// Отправляем имя файла
	filename := filepath.Base(filePath)
	err = stream.Send(&pb.UploadRequest{
		Data: &pb.UploadRequest_Filename{
			Filename: filename,
		},
	})
	if err != nil {
		log.Fatalf("Не удалось отправить имя файла: %v", err)
	}

	// Отправляем файл частями
	buffer := make([]byte, 64*1024) // 64KB chunks
	for {
		n, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Ошибка чтения файла: %v", err)
		}

		err = stream.Send(&pb.UploadRequest{
			Data: &pb.UploadRequest_Chunk{
				Chunk: buffer[:n],
			},
		})
		if err != nil {
			log.Fatalf("Не удалось отправить chunk: %v", err)
		}
		fmt.Printf("Отправлено %d байт\n", n)
	}

	// Получаем ответ
	response, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatalf("Ошибка при получении ответа: %v", err)
	}

	fmt.Printf("✅ Файл успешно загружен!\n")
	fmt.Printf("   Имя: %s\n", response.Filename)
	fmt.Printf("   Размер: %d байт\n", response.Size)
}

func listFiles(client pb.FileServiceClient) {
	fmt.Println("Получаем список файлов...")

	response, err := client.ListFiles(context.Background(), &pb.ListRequest{})
	if err != nil {
		log.Fatalf("Ошибка при получении списка: %v", err)
	}

	if len(response.Files) == 0 {
		fmt.Println("📁 Файлов нет")
		return
	}

	fmt.Println("\n📁 Список файлов:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("%-30s %-20s %-20s\n", "Имя файла", "Дата создания", "Дата обновления")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	for _, file := range response.Files {
		// Парсим дату для красивого отображения
		createdAt, _ := time.Parse(time.RFC3339, file.CreatedAt)
		updatedAt, _ := time.Parse(time.RFC3339, file.UpdatedAt)

		fmt.Printf("%-30s %-20s %-20s\n",
			file.Filename,
			createdAt.Format("2006-01-02 15:04:05"),
			updatedAt.Format("2006-01-02 15:04:05"))
	}
	fmt.Printf("\n📊 Всего файлов: %d\n", len(response.Files))
}

func downloadFile(client pb.FileServiceClient, filename string) {
	fmt.Printf("Скачиваем файл: %s\n", filename)

	// Создаем stream для скачивания
	stream, err := client.DownloadFile(context.Background(), &pb.DownloadRequest{
		Filename: filename,
	})
	if err != nil {
		log.Fatalf("Не удалось создать stream для скачивания: %v", err)
	}

	// Создаем файл для записи
	outputPath := "downloaded_" + filename
	outputFile, err := os.Create(outputPath)
	if err != nil {
		log.Fatalf("Не удалось создать выходной файл: %v", err)
	}
	defer outputFile.Close()

	var totalBytes int64 = 0

	// Получаем файл частями
	for {
		response, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Ошибка при получении chunk: %v", err)
		}

		n, err := outputFile.Write(response.Chunk)
		if err != nil {
			log.Fatalf("Ошибка при записи в файл: %v", err)
		}

		totalBytes += int64(n)
		fmt.Printf("Получено %d байт\n", n)
	}

	fmt.Printf("✅ Файл успешно скачан!\n")
	fmt.Printf("   Сохранен как: %s\n", outputPath)
	fmt.Printf("   Размер: %d байт\n", totalBytes)
}
