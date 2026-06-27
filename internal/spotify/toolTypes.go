package spotify

// TOON Types

// Artist is a simplified representation of a Spotify artist.
type Artist struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Album is a simplified representation of a Spotify album.
type Album struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Artists     string `json:"artists"`
	TotalTracks int    `json:"total_tracks"`
	ReleaseDate string `json:"release_date"`
}

// Track is a simplified representation of a Spotify track.
type Track struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Artists     string `json:"artists"`
	Album       string `json:"album"`
	DurationMs  int    `json:"duration_ms"`
	TrackNumber int    `json:"track_number"`
}

// Playlist is a simplified representation of a Spotify playlist.
type Playlist struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Owner         string `json:"owner"`
	Collaborative bool   `json:"collaborative"`
	Description   string `json:"description"`
}
