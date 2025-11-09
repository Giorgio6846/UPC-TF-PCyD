package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Movie struct {
	ID     int32    `bson:"_id"`
	IMDB   int32    `bson:"imdb,omitempty"`
	TMDB   int32    `bson:"tmdb,omitempty"`
	Title  string   `bson:"title"`
	Genres []string `bson:"genres,omitempty"`
}

type Ratings struct {
	ID      primitive.ObjectID `bson:"_id,omitempty"`
	UserID  int32              `bson:"userId"`
	MovieID int32              `bson:"movieId"`
	Rating  float64            `bson:"rating"`
}

type Tags struct {
	ID      primitive.ObjectID `bson:"_id,omitempty"`
	UserID  int32              `bson:"userId"`
	MovieID int32              `bson:"movieId"`
	Tag     string             `bson:"tag"`
}

type linksParsed struct {
	movieId int
	imdb    int32
	tmdb    int32
}

type RowDecoder[T any] func([]string) (T, error)

func parseCSV[T any](path string, decode RowDecoder[T]) ([]T, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %q: %w", path, err)
	}
	defer file.Close()

	r := csv.NewReader(file)

	hdr, err := r.Read()
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("read header: %w", err)
	}
	if len(hdr) > 0 {
		hdr[0] = strings.TrimPrefix(hdr[0], "\uFEFF")
	}

	var out []T
	line := 2
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("read row: %w", err)
		}

		if len(row) == 0 || (len(row) == 1 && strings.TrimSpace(row[0]) == "") {
			line++
			continue
		}

		for i := range row {
			row[i] = strings.TrimSpace(row[i])
		}

		v, err := decode(row)
		if err != nil {
			return nil, fmt.Errorf("Decode row %v: %w", row, err)
		}
		out = append(out, v)
		line++
	}
	return out, nil
}

func decodeMovie(row []string) (Movie, error) {
	if len(row) < 3 {
		return Movie{}, fmt.Errorf("want 5 cols, got %d", len(row))
	}

	id, err := strconv.Atoi(row[0])
	if err != nil {
		return Movie{}, fmt.Errorf("id: %w", err)
	}

	title := row[1]

	genre := []string(nil)
	if row[2] != "" {
		genre = strings.Split(row[2], "|")
	}

	return Movie{
		ID: int32(id), IMDB: 0, TMDB: 0, Title: title, Genres: genre,
	}, nil
}

func decodeLinks(row []string) (linksParsed, error) {
	if len(row) < 3 {
		return linksParsed{}, fmt.Errorf("want 3 cols, got %d", len(row))
	}

	movieId, err := strconv.Atoi(row[0])
	if err != nil {
		return linksParsed{}, fmt.Errorf("movieId: %w", err)
	}

	Imdb, err := strconv.Atoi(row[1])
	if err != nil {
		return linksParsed{}, fmt.Errorf("imdb: %w", err)
	}

	Tmdb, err := strconv.Atoi(row[2])
	if err != nil {
		Tmdb = 0
		//return links{}, fmt.Errorf("tmdb: %w", err)
	}

	return linksParsed{
		movieId: movieId, imdb: int32(Imdb), tmdb: int32(Tmdb),
	}, nil
}

func decodeRating(row []string) (Ratings, error) {
	if len(row) < 3 {
		return Ratings{}, fmt.Errorf("want 3 cols, got %d", len(row))
	}

	userID, err := strconv.Atoi(row[0])
	if err != nil {
		return Ratings{}, fmt.Errorf("userId: %w", err)
	}

	movieId, err := strconv.Atoi(row[1])
	if err != nil {
		return Ratings{}, fmt.Errorf("movieId: %w", err)
	}

	rv, err := strconv.ParseFloat(row[2], 32)
	if err != nil {
		return Ratings{}, fmt.Errorf("ratingId: %w", err)
	}

	return Ratings{
		UserID: int32(userID), MovieID: int32(movieId), Rating: rv,
	}, nil
}

func decodeTags(row []string) (Tags, error) {
	if len(row) < 3 {
		return Tags{}, fmt.Errorf("want 3 cols, got %d", len(row))
	}

	userId, err := strconv.Atoi(row[0])
	if err != nil {
		return Tags{}, fmt.Errorf("id: %w", err)
	}

	movieId, err := strconv.Atoi(row[1])
	if err != nil {
		return Tags{}, fmt.Errorf("id: %w", err)
	}

	tag := row[2]

	return Tags{
		UserID: int32(userId), MovieID: int32(movieId), Tag: tag,
	}, nil
}
