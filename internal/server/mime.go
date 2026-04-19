package server

import (
	"mime"
	"path"
	"strings"
)

// mimeByExt is consulted before the stdlib's mime.TypeByExtension because the
// stdlib implementation on Windows is registry-dependent and frequently misses
// common media formats (e.g. .mkv, .m4v, .ogv, .flac, .opus).
var mimeByExt = map[string]string{
	// Video
	".mp4":  "video/mp4",
	".m4v":  "video/mp4",
	".webm": "video/webm",
	".ogv":  "video/ogg",
	".mov":  "video/quicktime",
	".mkv":  "video/x-matroska",
	".avi":  "video/x-msvideo",
	".mpg":  "video/mpeg",
	".mpeg": "video/mpeg",
	".ts":   "video/mp2t",

	// Audio
	".mp3":  "audio/mpeg",
	".wav":  "audio/wav",
	".ogg":  "audio/ogg",
	".oga":  "audio/ogg",
	".flac": "audio/flac",
	".m4a":  "audio/mp4",
	".aac":  "audio/aac",
	".opus": "audio/ogg",

	// Image
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".gif":  "image/gif",
	".webp": "image/webp",
	".avif": "image/avif",
	".svg":  "image/svg+xml",
	".bmp":  "image/bmp",
	".ico":  "image/x-icon",

	// Text / docs
	".txt":  "text/plain; charset=utf-8",
	".log":  "text/plain; charset=utf-8",
	".md":   "text/plain; charset=utf-8",
	".json": "application/json",
	".csv":  "text/csv; charset=utf-8",
	".yaml": "text/yaml; charset=utf-8",
	".yml":  "text/yaml; charset=utf-8",
	".html": "text/html; charset=utf-8",
	".htm":  "text/html; charset=utf-8",
	".xml":  "application/xml",
	".pdf":  "application/pdf",
}

// contentTypeFor returns the MIME type for displayName, preferring the
// package-local table over mime.TypeByExtension. Returns "" if neither knows.
func contentTypeFor(displayName string) string {
	ext := strings.ToLower(path.Ext(displayName))
	if ext == "" {
		return ""
	}
	if ct, ok := mimeByExt[ext]; ok {
		return ct
	}
	return mime.TypeByExtension(ext)
}
