package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/PivKeyU/Emotion/internal/config"
	"github.com/PivKeyU/Emotion/internal/db"
)

// Catalog serves metadata catalog endpoints (Genres, Tags, Years, Studios).
type Catalog struct {
	db  *db.DB
	cfg *config.Config
	log *slog.Logger
}

// NewCatalog builds the handler.
func NewCatalog(database *db.DB, cfg *config.Config, log *slog.Logger) *Catalog {
	return &Catalog{
		db:  database,
		cfg: cfg,
		log: log,
	}
}

// Genres returns list of genres with counts.
func (c *Catalog) Genres(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	rows, err := c.db.QueryContext(ctx, `
		SELECT DISTINCT UNNEST(string_to_array(COALESCE(genre_str, ''), ',')) as genre
		FROM video_list
		WHERE deleted_at IS NULL AND genre_str IS NOT NULL AND genre_str != ''
		ORDER BY genre ASC
	`)
	if err != nil {
		c.log.Error("genres query failed", "err", err)
		WriteStatus(w, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	items := []map[string]any{}
	for rows.Next() {
		var genre string
		if err := rows.Scan(&genre); err != nil {
			continue
		}
		if genre = trimStr(genre); genre != "" {
			items = append(items, map[string]any{
				"Name": genre,
				"Id":   genre,
			})
		}
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"Items": items,
		"TotalRecordCount": len(items),
	})
}

// Tags returns list of tags with counts.
func (c *Catalog) Tags(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	rows, err := c.db.QueryContext(ctx, `
		SELECT DISTINCT UNNEST(string_to_array(COALESCE(tag_str, ''), ',')) as tag
		FROM video_list
		WHERE deleted_at IS NULL AND tag_str IS NOT NULL AND tag_str != ''
		ORDER BY tag ASC
	`)
	if err != nil {
		c.log.Error("tags query failed", "err", err)
		WriteStatus(w, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	items := []map[string]any{}
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			continue
		}
		if tag = trimStr(tag); tag != "" {
			items = append(items, map[string]any{
				"Name": tag,
				"Id":   tag,
			})
		}
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"Items": items,
		"TotalRecordCount": len(items),
	})
}

// Years returns list of production years.
func (c *Catalog) Years(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	rows, err := c.db.QueryContext(ctx, `
		SELECT DISTINCT EXTRACT(YEAR FROM date_air)::int as year
		FROM video_list
		WHERE deleted_at IS NULL AND date_air IS NOT NULL
		ORDER BY year DESC
	`)
	if err != nil {
		c.log.Error("years query failed", "err", err)
		WriteStatus(w, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	items := []map[string]any{}
	for rows.Next() {
		var year int
		if err := rows.Scan(&year); err != nil {
			continue
		}
		if year > 1900 {
			items = append(items, map[string]any{
				"Name": itoa(int64(year)),
				"Id":   itoa(int64(year)),
			})
		}
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"Items": items,
		"TotalRecordCount": len(items),
	})
}

// Studios returns list of studios.
func (c *Catalog) Studios(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	rows, err := c.db.QueryContext(ctx, `
		SELECT DISTINCT UNNEST(string_to_array(COALESCE(studio_str, ''), ',')) as studio
		FROM video_list
		WHERE deleted_at IS NULL AND studio_str IS NOT NULL AND studio_str != ''
		ORDER BY studio ASC
	`)
	if err != nil {
		c.log.Error("studios query failed", "err", err)
		WriteStatus(w, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	items := []map[string]any{}
	for rows.Next() {
		var studio string
		if err := rows.Scan(&studio); err != nil {
			continue
		}
		if studio = trimStr(studio); studio != "" {
			items = append(items, map[string]any{
				"Name": studio,
				"Id":   studio,
			})
		}
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"Items": items,
		"TotalRecordCount": len(items),
	})
}

func trimStr(s string) string {
	return stringTrimSpace(s)
}
