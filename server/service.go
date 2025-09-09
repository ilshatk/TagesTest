package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	pb "grpcTages/pbs" // Импорт сгенерированного кода
)

type fileServiceServer struct {
	pb.UnimplementedFileServiceServer
	uploadDir string
}

func NewFileServiceServer(uploadDir string) *fileServiceServer {
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Fatalf("Failed to create upload directory: %v", err)
	}
	return &fileServiceServer{uploadDir: uploadDir}
}

func (s *fileServiceServer) UploadFile(stream pb.FileService_UploadFileServer) error {
	// Семафор для ограничения конкурентных загрузок
	uploadSem <- struct{}{}
	defer func() { <-uploadSem }()

	var filename string
	var file *os.File
	defer func() {
		if file != nil {
			file.Close()
		}
	}()

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch data := req.Data.(type) {
		case *pb.UploadRequest_Filename:
			filename = data.Filename
			filePath := filepath.Join(s.uploadDir, filename)
			file, err = os.Create(filePath)
			if err != nil {
				return fmt.Errorf("failed to create file: %v", err)
			}
		case *pb.UploadRequest_Chunk:
			if file == nil {
				return fmt.Errorf("filename not received before data")
			}
			if _, err := file.Write(data.Chunk); err != nil {
				return fmt.Errorf("failed to write chunk: %v", err)
			}
		}
	}

	// Получаем информацию о файле для ответа
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %v", err)
	}

	return stream.SendAndClose(&pb.UploadResponse{
		Filename: filename,
		Size:     uint32(fileInfo.Size()),
	})
}

func (s *fileServiceServer) ListFiles(ctx context.Context, req *pb.ListRequest) (*pb.ListResponse, error) {
	// Семафор для ограничения конкурентных запросов списка
	listSem <- struct{}{}
	defer func() { <-listSem }()

	files, err := os.ReadDir(s.uploadDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read upload directory: %v", err)
	}

	var fileInfos []*pb.FileInfo
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue // Пропускаем файлы с ошибками
		}

		fileInfos = append(fileInfos, &pb.FileInfo{
			Filename:  file.Name(),
			CreatedAt: info.ModTime().Format(time.RFC3339),
			UpdatedAt: info.ModTime().Format(time.RFC3339),
		})
	}

	return &pb.ListResponse{Files: fileInfos}, nil
}

func (s *fileServiceServer) DownloadFile(req *pb.DownloadRequest, stream pb.FileService_DownloadFileServer) error {
	// Семафор для ограничения конкурентных скачиваний
	uploadSem <- struct{}{}
	defer func() { <-uploadSem }()

	filePath := filepath.Join(s.uploadDir, req.Filename)
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	buffer := make([]byte, 64*1024) // 64KB chunks
	for {
		n, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read file: %v", err)
		}

		if err := stream.Send(&pb.DownloadResponse{
			Chunk: buffer[:n],
		}); err != nil {
			return fmt.Errorf("failed to send chunk: %v", err)
		}
	}

	return nil
}

// Глобальные семафоры для ограничения конкурентных запросов
var (
	uploadSem = make(chan struct{}, 10)  // 10 одновременных загрузок/скачиваний
	listSem   = make(chan struct{}, 100) // 100 одновременных запросов списка
)
