# gRPC File Service - Краткая инструкция

### Запустить сервис
```
Запустить server.exe
```

### Загрузить изображение:
```bash
go run client.go upload C:\path\to\image.jpg
```

### Загрузить документ:
```bash
go run client.go upload document.pdf
```

### Посмотреть все файлы:
```bash
go run client.go list
```

### Скачать конкретный файл:
```bash
go run client.go download image.jpg
```
