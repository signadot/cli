package artifact

import "github.com/go-openapi/runtime"

func getDownloadConsumers() map[string]runtime.Consumer {
	return map[string]runtime.Consumer{
		"image/png":          runtime.ByteStreamConsumer(),
		"image/jpeg":         runtime.ByteStreamConsumer(),
		"image/gif":          runtime.ByteStreamConsumer(),
		"image/bmp":          runtime.ByteStreamConsumer(),
		"image/webp":         runtime.ByteStreamConsumer(),
		"application/pdf":    runtime.ByteStreamConsumer(),
		"application/msword": runtime.ByteStreamConsumer(),
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": runtime.ByteStreamConsumer(),
		"application/vnd.ms-excel": runtime.ByteStreamConsumer(),
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         runtime.ByteStreamConsumer(),
		"application/vnd.ms-powerpoint":                                             runtime.ByteStreamConsumer(),
		"application/vnd.openxmlformats-officedocument.presentationml.presentation": runtime.ByteStreamConsumer(),
		"application/zip":              runtime.ByteStreamConsumer(),
		"application/x-rar-compressed": runtime.ByteStreamConsumer(),
		"application/x-tar":            runtime.ByteStreamConsumer(),
		"application/x-7z-compressed":  runtime.ByteStreamConsumer(),
		"application/json":             runtime.ByteStreamConsumer(),
		"text/plain":                   runtime.ByteStreamConsumer(),
		"text/html":                    runtime.ByteStreamConsumer(),
		"text/css":                     runtime.ByteStreamConsumer(),
		"text/javascript":              runtime.ByteStreamConsumer(),
		"application/javascript":       runtime.ByteStreamConsumer(),
		"application/xml":              runtime.ByteStreamConsumer(),
		"audio/mpeg":                   runtime.ByteStreamConsumer(),
		"audio/ogg":                    runtime.ByteStreamConsumer(),
		"audio/wav":                    runtime.ByteStreamConsumer(),
		"audio/webm":                   runtime.ByteStreamConsumer(),
		"video/mp4":                    runtime.ByteStreamConsumer(),
		"video/mpeg":                   runtime.ByteStreamConsumer(),
		"video/ogg":                    runtime.ByteStreamConsumer(),
		"video/webm":                   runtime.ByteStreamConsumer(),
		"video/x-msvideo":              runtime.ByteStreamConsumer(),
	}
}
