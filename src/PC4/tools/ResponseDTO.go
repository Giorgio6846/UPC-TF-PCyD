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
