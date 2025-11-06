package main

type movie struct {
	id     int
	imdb   int
	tmdb   int
	title  string
	genres []string
}

type rating struct {
	userId  int
	movieId int
	rating  float32
}

type tags struct {
	userId  int
	movieId int
	tag     string
}

func dataParser() {

}
