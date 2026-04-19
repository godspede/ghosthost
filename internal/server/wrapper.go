package server

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/godspede/ghosthost/internal/share"
)

// videoTpl lets the <video> element size naturally (max-width/max-height, not
// width/height:100vh) so the native control bar renders immediately below the
// visible frame inside the element. On portrait mobile viewing landscape
// video, letterboxing happens inside the element, not inside the viewport —
// this keeps controls above iOS Safari's bottom chrome.
var videoTpl = template.Must(template.New("video").Parse(`<!doctype html><meta charset=utf-8>
<title>{{.Name}}</title>
<style>
  html,body{margin:0;height:100%;background:#000;}
  main{min-height:100vh;min-height:100dvh;display:flex;align-items:center;justify-content:center;}
  video{display:block;max-width:100vw;max-height:100vh;max-height:100dvh;background:#000;}
</style>
<main><video src="?raw=1" controls muted autoplay playsinline></video></main>
`))

// audioTpl intentionally omits `muted`: the user opened an audio clip to hear
// it. If a browser blocks autoplay without muted, the user taps play.
var audioTpl = template.Must(template.New("audio").Parse(`<!doctype html><meta charset=utf-8>
<title>{{.Name}}</title>
<style>
  html,body{margin:0;height:100%;}
  body{background:#f4f4f5;color:#333;font:14px system-ui,sans-serif;}
  main{min-height:100vh;min-height:100dvh;display:flex;flex-direction:column;align-items:center;justify-content:center;gap:16px;padding:24px 16px;box-sizing:border-box;}
  .name{max-width:100%;overflow-wrap:anywhere;text-align:center;}
  audio{width:100%;max-width:560px;}
</style>
<main>
  <div class="name">{{.Name}}</div>
  <audio src="?raw=1" controls autoplay preload="auto"></audio>
</main>
`))

func isVideo(displayName string) bool {
	return strings.HasPrefix(contentTypeFor(displayName), "video/")
}

func isAudio(displayName string) bool {
	return strings.HasPrefix(contentTypeFor(displayName), "audio/")
}

// wantsMediaWrapper returns true when we should serve an HTML wrapper for a
// browser navigating to a media URL. Raw / Range / download requests bypass.
func wantsMediaWrapper(r *http.Request) bool {
	if r.URL.Query().Get("raw") == "1" || r.URL.Query().Get("dl") == "1" {
		return false
	}
	if r.Header.Get("Range") != "" {
		return false
	}
	return strings.Contains(r.Header.Get("Accept"), "text/html")
}

func wantsVideoWrapper(r *http.Request, displayName string) bool {
	return isVideo(displayName) && wantsMediaWrapper(r)
}

func wantsAudioWrapper(r *http.Request, displayName string) bool {
	return isAudio(displayName) && wantsMediaWrapper(r)
}

func writeWrapperHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
}

func serveVideoWrapperReal(w http.ResponseWriter, r *http.Request, s *share.Share) bool {
	if !wantsVideoWrapper(r, s.DisplayName) {
		return false
	}
	writeWrapperHeaders(w)
	_ = videoTpl.Execute(w, struct{ Name string }{Name: s.DisplayName})
	return true
}

func serveAudioWrapperReal(w http.ResponseWriter, r *http.Request, s *share.Share) bool {
	if !wantsAudioWrapper(r, s.DisplayName) {
		return false
	}
	writeWrapperHeaders(w)
	_ = audioTpl.Execute(w, struct{ Name string }{Name: s.DisplayName})
	return true
}
