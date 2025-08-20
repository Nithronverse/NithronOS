package apps

import (
	"os"
	"path/filepath"
)

type App struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

var defaultApps = []App{
	{ID: "plex", Name: "Plex"},
	{ID: "jellyfin", Name: "Jellyfin"},
	{ID: "qbittorrent", Name: "qBittorrent"},
	{ID: "nextcloud", Name: "Nextcloud"},
	{ID: "immich", Name: "Immich"},
}

func Catalog(installDir string) []App {
	out := make([]App, 0, len(defaultApps))
	for _, a := range defaultApps {
		dir := filepath.Join(installDir, a.ID)
		if fi, err := os.Stat(dir); err == nil && fi.IsDir() {
			a.Status = "installed"
		} else {
			a.Status = "not_installed"
		}
		out = append(out, a)
	}
	return out
}

// ComposeTemplate returns a minimal docker-compose.yml for the given app id.
func ComposeTemplate(id string) string {
	switch id {
	case "jellyfin":
		return `services:
  jellyfin:
    image: jellyfin/jellyfin:latest
    restart: unless-stopped
    network_mode: host
    volumes:
      - ./config:/config
      - ./cache:/cache
      - /srv/media:/media:ro
`
	case "plex":
		return `services:
  plex:
    image: lscr.io/linuxserver/plex:latest
    restart: unless-stopped
    network_mode: host
    environment:
      - PUID=1000
      - PGID=1000
    volumes:
      - ./config:/config
      - /srv/media:/media:ro
`
	case "qbittorrent":
		return `services:
  qbittorrent:
    image: lscr.io/linuxserver/qbittorrent:latest
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      - PUID=1000
      - PGID=1000
    volumes:
      - ./config:/config
      - /srv/downloads:/downloads
`
	case "nextcloud":
		return `services:
  nextcloud:
    image: lscr.io/linuxserver/nextcloud:latest
    restart: unless-stopped
    ports:
      - "8081:443"
    environment:
      - PUID=1000
      - PGID=1000
    volumes:
      - ./config:/config
      - ./data:/data
`
	case "immich":
		return `services:
  immich-server:
    image: ghcr.io/immich-app/immich-server:release
    restart: unless-stopped
    ports:
      - "2283:2283"
    volumes:
      - ./library:/usr/src/app/upload
`
	default:
		return `services:
  app:
    image: alpine:latest
    command: ["sleep","infinity"]
`
	}
}

func UnitTemplate(id, dir string) string {
	return `[Unit]
Description=NithronOS App (` + id + `)
After=docker.service
Requires=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=` + dir + `
ExecStart=/usr/bin/docker compose up -d
ExecStop=/usr/bin/docker compose down
TimeoutStartSec=0

[Install]
WantedBy=multi-user.target
`
}
