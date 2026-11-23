package webassets

import "embed"

// Assets holds all embedded static Bootstrap CSS, favicon, and HTML5 templates.
//
//go:embed static/* templates/*.html
var Assets embed.FS
