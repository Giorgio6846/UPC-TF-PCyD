import React, { useEffect, useState } from "react";
import { useAuth } from "../context/AuthContext";
import { fetchRecommendedMovies, fetchTmdbImages } from "../api/client";
import type { JsonMovieResult, ResponseMovieJSON } from "../api/client";

interface MovieWithImages extends JsonMovieResult {
  posterUrl?: string | null;
  backdropUrl?: string | null;
}

const UserDashboard: React.FC = () => {
  const { token, email, displayName } = useAuth();
  const [userId, setUserId] = useState("1");
  const [data, setData] = useState<ResponseMovieJSON | null>(null);
  const [movies, setMovies] = useState<MovieWithImages[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const handleFetch = async () => {
    if (!token) {
      setError("Necesitas iniciar sesion primero.");
      return;
    }
    if (!userId.trim()) {
      setError("Ingresa un userId de MovieLens.");
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const res = await fetchRecommendedMovies(token, Number(userId));
      setData(res);
      setMovies(res.results);
    } catch (err: any) {
      setData(null);
      setMovies([]);
      setError(err?.message ?? "Error obteniendo recomendaciones.");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    const loadPosters = async () => {
      if (!data?.results?.length) return;
      const entries = await Promise.all(
        data.results.map(async movie => {
          const { posterUrl, backdropUrl } = await fetchTmdbImages(movie.tmdb);
          return { ...movie, posterUrl, backdropUrl } as MovieWithImages;
        })
      );
      setMovies(entries);
    };
    loadPosters();
  }, [data]);

  const hasMovies = movies?.length;

  return (
    <section className="page dashboard">
      <div className="page-header">
        <div>
          <p className="pill">Panel de recomendaciones</p>
          <h1>Hola {displayName || email || "explorador"}</h1>
          <p className="muted">Ingresa un userId de MovieLens para ver el top de peliculas personalizadas.</p>
        </div>
      </div>

      <div className="panel form-panel">
        <div className="form-row">
          <label className="input-group inline">
            <span>User ID</span>
            <input
              type="number"
              min="1"
              value={userId}
              onChange={e => setUserId(e.target.value)}
              placeholder="Ej. 42"
            />
          </label>
          <button className="btn primary" onClick={handleFetch} disabled={loading}>
            {loading ? "Consultando..." : "Obtener recomendaciones"}
          </button>
        </div>
        <p className="micro">Tip: el dataset MovieLens 1M tiene usuarios entre 1 y 600.</p>
        {error && <div className="error-banner">{error}</div>}
      </div>

      {data ? (
        <>
          <div className="stat-grid">
            <div className="stat-card">
              <p className="muted">Consulta DB</p>
              <strong>{data.durationDB} ms</strong>
            </div>
            <div className="stat-card">
              <p className="muted">Algoritmo</p>
              <strong>{data.durationAlgo} ms</strong>
            </div>
            <div className="stat-card">
              <p className="muted">Detalle peliculas</p>
              <strong>{data.durationMovieFetch} ms</strong>
            </div>
          </div>

          {hasMovies ? (
            <div className="movies-grid">
              {movies.map(movie => {
                const cleanedGenres = movie.genres?.filter(
                  g => g && g.toLowerCase() !== "(no genres listed)"
                );
                const genresLabel = cleanedGenres && cleanedGenres.length ? cleanedGenres.join(" / ") : "-";
                const heroStyle = movie.backdropUrl
                  ? {
                      background:
                        `linear-gradient(180deg, rgba(3, 7, 18, 0.65), rgba(3, 7, 18, 0.85)), url(${movie.backdropUrl})`
                    }
                  : undefined;
                const heroClassName = movie.backdropUrl ? "movie-hero with-backdrop" : "movie-hero";
                return (
                  <article key={movie.movieId} className="movie-card">
                    <div className="movie-rank">#{movie.rank}</div>
                    <div className={heroClassName} style={heroStyle}>
                      {movie.posterUrl ? (
                        <img src={movie.posterUrl} alt={movie.title} className="movie-poster" loading="lazy" />
                      ) : (
                        <div className="poster-fallback">
                          <span>{movie.title.slice(0, 1)}</span>
                        </div>
                      )}
                    </div>
                    <div className="movie-head">
                      <h3>{movie.title}</h3>
                      <span className="score-chip">Score {movie.score.toFixed(3)}</span>
                    </div>
                    <p className="muted">{genresLabel}</p>
                    <div className="ids">
                      {movie.imdb && <span>IMDb {movie.imdb}</span>}
                      {movie.tmdb && <span>TMDb {movie.tmdb}</span>}
                    </div>
                  </article>
                );
              })}
            </div>
          ) : (
            <div className="panel empty-panel">
              <h3>Sin peliculas</h3>
              <p className="muted">No hay resultados para este usuario.</p>
            </div>
          )}
        </>
      ) : (
        !loading && (
          <div className="panel empty-panel">
            <h3>Consulta un usuario</h3>
            <p className="muted">Lanza la peticion para ver el top de peliculas recomendado.</p>
          </div>
        )
      )}
    </section>
  );
};

export default UserDashboard;
