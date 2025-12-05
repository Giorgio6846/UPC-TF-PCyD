package tools

type JsonMovieResult struct {
	Rank    int      `json:"rank"`
	MovieID int      `json:"movieId"`
	Title   string   `json:"title"`
	Genres  []string `json:"genres"`
	IMDB    int32    `json:"imdb,omitempty"`
	TMDB    int32    `json:"tmdb,omitempty"`
	Score   float64  `json:"score"`
}

type ResponseMovieJSON struct {
	Results            []JsonMovieResult `json:"results"`
	DurationDB         float32           `json:"durationDB"`
	DurationAlgo       float32           `json:"durationAlgo"`
	DurationMovieFetch float32           `json:"durationMovieFetch"`
}

type ResponseUserID struct {
	UserID   int    `json:"userId"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	LastName string `json:"lastName"`
}

type ResponseUserMovies struct {
	UserID       int            `json:"userId"`
	MovieRatings []MovieRatings `json:"moviesRatings"`
}

type ResponseSimlarities struct {
	SimilarUsers []Similarity `json:"similarity"`
	DurationDB   float32      `json:"durationDB"`
	DurationAlgo float32      `json:"durationAlgo"`
}

type ResourcesResponse struct {
	CPUPercent        float64  `json:"cpu_percent"`
	MemoryTotalKB     uint64   `json:"memory_total_kb"`
	MemoryAvailableKB uint64   `json:"memory_available_kb"`
	MemoryUsedKB      uint64   `json:"memory_used_kb"`
	MemoryUsedPercent float64  `json:"memory_used_percent"`
	NetworkBytesSent  uint64   `json:"network_bytes_sent"`
	NetworkBytesRecv  uint64   `json:"network_bytes_recv"`
	Workers           []string `json:"workers"`
	TimestampMs       int64    `json:"timestamp_ms"`
}
