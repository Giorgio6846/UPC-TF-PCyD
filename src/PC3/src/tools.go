package main

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func indexLinks(ls []linksParsed) map[int]linksParsed {
	idx := make(map[int]linksParsed, len(ls))
	for _, l := range ls {
		idx[l.movieId] = l
	}
	return idx
}

func fillMovieWithLinks(ms []Movie, idx map[int]linksParsed) {
	for i := range ms {
		if l, ok := idx[int(ms[i].ID)]; ok {
			ms[i].IMDB = l.imdb
			ms[i].TMDB = l.tmdb
		}
	}
}

func anySlice[T any](in []T) []interface{} {
	out := make([]interface{}, len(in))
	for i := range in {
		out[i] = in[i]
	}
	return out
}

func insertManyBatched(ctx context.Context, coll *mongo.Collection, docs []interface{}, batchSize int) error {
	if len(docs) == 0 {
		return nil
	}
	opts := options.InsertMany().SetOrdered(false) // continue on dup key
	total := 0
	for start := 0; start < len(docs); start += batchSize {
		end := start + batchSize
		if end > len(docs) {
			end = len(docs)
		}
		batch := docs[start:end]
		_, err := coll.InsertMany(ctx, batch, opts)
		if err != nil {
			// Allow duplicate key bulk errors (common on reruns)
			var bwe mongo.BulkWriteException
			if errors.As(err, &bwe) {
				// If *all* errors are duplicate key, ignore; otherwise return
				nonDup := false
				for _, we := range bwe.WriteErrors {
					if we.Code != 11000 { // duplicate key
						nonDup = true
						break
					}
				}
				if nonDup {
					return fmt.Errorf("bulk write error: %w", err)
				}
				// else: ignore dupes and continue
			} else {
				return err
			}
		}
		total += (end - start)
	}
	fmt.Printf("Inserted (attempted) %d into %s\n", total, coll.Name())
	return nil
}
