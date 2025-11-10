package main

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
