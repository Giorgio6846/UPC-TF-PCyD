package database

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"pc4/tools"
	"strconv"
	"strings"
	"sync"
)

type RowDecoder[T any] func([]string) (T, error)

type decodeJob[T any] struct {
	idx int
	row []string
}

type decodeResult[T any] struct {
	idx int
	val T
	err error
}

func parseCSV[T any](path string, decode RowDecoder[T], workers int) ([]T, error) {
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

	jobs := make(chan decodeJob[T])
	results := make(chan decodeResult[T])

	var wg sync.WaitGroup

	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				for i := range job.row {
					job.row[i] = strings.TrimSpace(job.row[i])
				}

				val, err := decode(job.row)
				results <- decodeResult[T]{
					idx: job.idx,
					val: val,
					err: err,
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	go func() {
		idx := 0
		for {
			row, err := r.Read()
			if err == io.EOF {
				break
			}

			if err != nil {
				results <- decodeResult[T]{err: fmt.Errorf("read row: %w", err)}
			}

			if len(row) == 0 || (len(row) == 1 && strings.TrimSpace(row[0]) == "") {
				continue
			}

			jobs <- decodeJob[T]{
				idx: idx,
				row: row,
			}
			idx++

		}
		close(jobs)
	}()

	var (
		out      []T
		firstErr error
	)

	for res := range results {
		if res.err != nil {
			if firstErr == nil {
				firstErr = res.err
			}
			continue
		}
		out = append(out, res.val)
	}

	if firstErr != nil {
		return nil, firstErr
	}
	return out, nil
}

func decodeMovie(row []string) (tools.Movie, error) {
	if len(row) < 3 {
		return tools.Movie{}, fmt.Errorf("want 5 cols, got %d", len(row))
	}

	id, err := strconv.Atoi(row[0])
	if err != nil {
		return tools.Movie{}, fmt.Errorf("id: %w", err)
	}

	title := row[1]

	genre := []string(nil)
	if row[2] != "" {
		genre = strings.Split(row[2], "|")
	}

	return tools.Movie{
		ID: int32(id), IMDB: 0, TMDB: 0, Title: title, Genres: genre,
	}, nil
}

func decodeLinks(row []string) (tools.LinksParsed, error) {
	if len(row) < 3 {
		return tools.LinksParsed{}, fmt.Errorf("want 3 cols, got %d", len(row))
	}

	movieId, err := strconv.Atoi(row[0])
	if err != nil {
		return tools.LinksParsed{}, fmt.Errorf("movieId: %w", err)
	}

	Imdb, err := strconv.Atoi(row[1])
	if err != nil {
		return tools.LinksParsed{}, fmt.Errorf("imdb: %w", err)
	}

	Tmdb, _ := strconv.Atoi(row[2])

	return tools.LinksParsed{
		MovieId: movieId, Imdb: int32(Imdb), Tmdb: int32(Tmdb),
	}, nil
}

func decodeRating(row []string) (tools.Ratings, error) {
	if len(row) < 3 {
		return tools.Ratings{}, fmt.Errorf("want 3 cols, got %d", len(row))
	}

	userID, err := strconv.Atoi(row[0])
	if err != nil {
		return tools.Ratings{}, fmt.Errorf("userId: %w", err)
	}

	movieId, err := strconv.Atoi(row[1])
	if err != nil {
		return tools.Ratings{}, fmt.Errorf("movieId: %w", err)
	}

	rv, err := strconv.ParseFloat(row[2], 32)
	if err != nil {
		return tools.Ratings{}, fmt.Errorf("ratingId: %w", err)
	}

	return tools.Ratings{
		UserID: int32(userID), MovieID: int32(movieId), Rating: rv,
	}, nil
}

func decodeTags(row []string) (tools.Tags, error) {
	if len(row) < 3 {
		return tools.Tags{}, fmt.Errorf("want 3 cols, got %d", len(row))
	}

	userId, err := strconv.Atoi(row[0])
	if err != nil {
		return tools.Tags{}, fmt.Errorf("id: %w", err)
	}

	movieId, err := strconv.Atoi(row[1])
	if err != nil {
		return tools.Tags{}, fmt.Errorf("id: %w", err)
	}

	tag := row[2]

	return tools.Tags{
		UserID: int32(userId), MovieID: int32(movieId), Tag: tag,
	}, nil
}
