// Package web содержит собранный Vite-фронтенд (каталог dist), вшиваемый в бинарник.
package web

import "embed"

//go:embed all:dist
var Dist embed.FS
