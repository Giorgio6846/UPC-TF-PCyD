import React, { useEffect, useState } from "react";
import { useAuth } from "../context/AuthContext";
import { fetchRecommendedMovies, fetchTmdbImages, fetchMe, fetchMyMovies, fetchSimilarUsers } from "../api/client";
import type { JsonMovieResult, ResponseMovieJSON, MyMovieEntry, SimilarUser } from "../api/client";

interface MovieWithImages extends JsonMovieResult {
  year?: number;
  posterUrl?: string | null;
  backdropUrl?: string | null;
}

interface MyMovieWithImages {
  id: number;
  title: string;
  genres: string[];
  rating: number;
  imdb?: number;
  tmdb?: number;
  posterUrl?: string | null;
  backdropUrl?: string | null;
}

const parseYearFromTitle = (title: string): number | undefined => {
  const match = title.match(/\((\d{4})\)/);
  return match ? Number(match[1]) : undefined;
};

const UserDashboard: React.FC = () => {
  const { token, email, displayName } = useAuth();
  
  const [data, setData] = useState<ResponseMovieJSON | null>(null);
  const [movies, setMovies] = useState<MovieWithImages[]>([]);
  const [myMovies, setMyMovies] = useState<MyMovieWithImages[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [myLoading, setMyLoading] = useState(false);
  const [similarUsers, setSimilarUsers] = useState<SimilarUser[]>([]);
  const [similarLoading, setSimilarLoading] = useState(false);
  const [chunks, setChunks] = useState<number>(50);
  const [currentUserId, setCurrentUserId] = useState<number | null>(null);
  const [sortBy, setSortBy] = useState<"score" | "year">("score");
  const [selectedGenre, setSelectedGenre] = useState<string>("all");
  const [activeTab, setActiveTab] = useState<"my" | "rec" | "similar">("rec");



  useEffect(() => {
    const loadPosters = async () => {
      if (!data?.results?.length) return;
      setLoading(true);
      const entries = await Promise.all(
        data.results.map(async movie => {
          const { posterUrl, backdropUrl } = await fetchTmdbImages(movie.tmdb, movie.imdb);
          return { ...movie, year: parseYearFromTitle(movie.title), posterUrl, backdropUrl } as MovieWithImages;
        })
      );
      setMovies(entries);
    };
    loadPosters();
  }, [data]);

  // Load posters for myMovies when they are set (only for entries missing images)
  useEffect(() => {
    const needImages = myMovies.some(m => m.posterUrl == null && m.backdropUrl == null);
    if (!myMovies.length || !needImages) return;
    let cancelled = false;
    const loadMyPosters = async () => {
      setMyLoading(true);
      const entries = await Promise.all(
        myMovies.map(async m => {
          const { posterUrl, backdropUrl } = await fetchTmdbImages(m.tmdb, m.imdb);
          return { ...m, posterUrl, backdropUrl } as MyMovieWithImages;
        })
      );
      if (!cancelled) setMyMovies(entries);
      setMyLoading(false);
    };
    loadMyPosters();
    return () => {
      cancelled = true;
    };
  }, [myMovies]);

  const availableGenres = React.useMemo(() => {
    const set = new Set<string>();
    movies.forEach(m => {
      m.genres?.forEach(g => {
        if (g && g.toLowerCase() !== "(no genres listed)") set.add(g);
      });
    });
    return Array.from(set).sort((a, b) => a.localeCompare(b));
  }, [movies]);

  const filteredMovies = React.useMemo(() => {
    const matchesGenre = (genres?: string[]) => {
      if (selectedGenre === "all") return true;
      return genres?.some(g => g === selectedGenre);
    };

    return movies
      .filter(m => matchesGenre(m.genres))
      .sort((a, b) => {
        if (sortBy === "year") {
          const ay = a.year ?? -Infinity;
          const by = b.year ?? -Infinity;
          if (ay === by) return b.score - a.score;
          return by - ay;
        }
        // score (default)
        return b.score - a.score;
      });
  }, [movies, sortBy, selectedGenre]);

  const hasMovies = filteredMovies?.length;
  const handleReload = async () => {
    if (!token) {
      setError("Necesitas iniciar sesion primero.");
      return;
    }
    if (!currentUserId) {
      setError("No hay userId disponible para recargar.");
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const res = await fetchRecommendedMovies(token, currentUserId, chunks);
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
  // Cuando hay token, obtener /me y lanzar la petición automáticamente
  useEffect(() => {
    if (!token) return;
    let cancelled = false;

    const loadMeAndRecommendations = async () => {
      setLoading(true);
      setError(null);
      setMyLoading(true);
      setSimilarLoading(true);
      try {
        const me = await fetchMe(token);
        if (cancelled) return;
        setCurrentUserId(Number(me.userId));

        // Fetch recommendations, user's movies and similar users in parallel
        const [res, myRes, simRes] = await Promise.all([
          fetchRecommendedMovies(token, Number(me.userId), chunks),
          fetchMyMovies(token),
          fetchSimilarUsers(token, Number(me.userId))
        ]);
        if (cancelled) return;
        setData(res);
        setMovies(res.results);
        if (myRes?.moviesRatings) {
          const mapped = myRes.moviesRatings.map((m: MyMovieEntry) => ({
            id: m.Movie.ID,
            title: m.Movie.Title,
            genres: m.Movie.Genres || [],
            rating: m.Rating,
            imdb: m.Movie.IMDB,
            tmdb: m.Movie.TMDB,
            posterUrl: null,
            backdropUrl: null
          } as MyMovieWithImages));
          setMyMovies(mapped);
        } else {
          setMyMovies([]);
        }
        // similar users handled above
        if (simRes?.similarity) {
          setSimilarUsers(simRes.similarity || []);
        } else {
          setSimilarUsers([]);
        }
      } catch (err: any) {
        if (cancelled) return;
        setData(null);
        setMovies([]);
        setError(err?.message ?? "Error obteniendo recomendaciones.");
        setMyMovies([]);
        setSimilarUsers([]);
      } finally {
        if (!cancelled) {
          setLoading(false);
          setMyLoading(false);
          setSimilarLoading(false);
        }
      }
    };

    loadMeAndRecommendations();

    return () => {
      cancelled = true;
    };
  }, [token]);

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
          {activeTab === "rec" ? (
            loading ? (
              <p className="muted">Cargando recomendaciones...</p>
            ) : (
              <p className="muted">Las recomendaciones se cargarán automáticamente al iniciar sesión.</p>
            )
          ) : activeTab === "my" ? (
            myLoading ? (
              <p className="muted">Cargando tus películas...</p>
            ) : (
              <p className="muted">Tus películas aparecen en esta pestaña.</p>
            )
          ) : (
            similarLoading ? (
              <p className="muted">Cargando usuarios similares...</p>
            ) : (
              <p className="muted">Consulta los usuarios similares en su pestaña.</p>
            )
          )}
        </div>
        {error && <div className="error-banner">{error}</div>}
        {data && activeTab === "rec" && (
          <div className="form-row filters-row" style={{ marginTop: "12px" }}>
            <label className="input-group">
              <span>Ordenar por</span>
              <select
                className="select-solid"
                value={sortBy}
                onChange={e => setSortBy(e.target.value as typeof sortBy)}
              >
                <option value="score">Score</option>
                <option value="year">Año</option>
              </select>
            </label>
            <label className="input-group">
              <span>Género</span>
              <select
                className="select-solid"
                value={selectedGenre}
                onChange={e => setSelectedGenre(e.target.value)}
              >
                <option value="all">Todos</option>
                {availableGenres.map(g => (
                  <option key={g} value={g}>
                    {g}
                  </option>
                ))}
              </select>
            </label>
            <label className="input-group inline">
              <span>Chunks</span>
              <input
                type="number"
                min={1}
                className="select-solid"
                value={chunks}
                onChange={e => setChunks(Number(e.target.value) || 0)}
              />
            </label>
            <div style={{ display: "flex", alignItems: "center" }}>
              <button className="btn" type="button" onClick={handleReload}>
                Recargar
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Tabs: Tus películas / Recomendaciones */}
      <div className="tabs" style={{ marginTop: 12, marginBottom: 12 }}>
        <button
          className={activeTab === "my" ? "tab active" : "tab"}
          onClick={() => setActiveTab("my")}
          type="button"
        >
          Tus películas
        </button>
        <button
          className={activeTab === "rec" ? "tab active" : "tab"}
          onClick={() => setActiveTab("rec")}
          type="button"
        >
          Recomendaciones
        </button>
        <button
          className={activeTab === "similar" ? "tab active" : "tab"}
          onClick={() => setActiveTab("similar")}
          type="button"
        >
          Usuarios similares
        </button>
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

          {activeTab === "my" ? (
            myLoading ? (
              <div className="panel empty-panel">
                <h3>Cargando tus películas</h3>
                <p className="muted">Espere mientras obtenemos tus películas valoradas.</p>
              </div>
            ) : myMovies && myMovies.length ? (
              <div className="movies-grid">
                {myMovies.map(m => {
                  const cleanedGenres = m.genres?.filter(
                    g => g && g.toLowerCase() !== "(no genres listed)"
                  );
                  const genresLabel = cleanedGenres && cleanedGenres.length ? cleanedGenres.join(" / ") : "-";
                  const heroStyle = m.backdropUrl
                    ? {
                        background:
                          `linear-gradient(180deg, rgba(3, 7, 18, 0.65), rgba(3, 7, 18, 0.85)), url(${m.backdropUrl})`
                      }
                    : undefined;
                  const heroClassName = m.backdropUrl ? "movie-hero with-backdrop" : "movie-hero";
                  return (
                    <article key={m.id} className="movie-card">
                      <div className="movie-rank">&nbsp;</div>
                      <div className={heroClassName} style={heroStyle}>
                        {m.posterUrl ? (
                          <img src={m.posterUrl} alt={m.title} className="movie-poster" loading="lazy" />
                        ) : (
                          <div className="poster-fallback">
                            <span>{m.title.slice(0, 1)}</span>
                          </div>
                        )}
                      </div>
                      <div className="movie-head">
                        <h3>{m.title}</h3>
                        <span className="score-chip">Puntuación {m.rating.toFixed(1)}</span>
                      </div>
                      <p className="muted">{genresLabel}</p>
                      <div className="ids">
                        {m.imdb && <span>IMDb {m.imdb}</span>}
                        {m.tmdb && (
                          <a href={`https://www.themoviedb.org/movie/${m.tmdb}`} target="_blank" rel="noreferrer">
                            TMDb {m.tmdb}
                          </a>
                        )}
                      </div>
                    </article>
                  );
                })}
              </div>
            ) : (
              <div className="panel empty-panel">
                <h3>Sin películas</h3>
                <p className="muted">No se encontraron películas en tu historial.</p>
              </div>
            )
          ) : activeTab === "rec" ? (
            hasMovies ? (
              <div className="movies-grid">
                {filteredMovies.map(movie => {
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
                        {movie.tmdb && (
                          <a
                            href={`https://www.themoviedb.org/movie/${movie.tmdb}`}
                            target="_blank"
                            rel="noreferrer"
                          >
                            TMDb {movie.tmdb}
                          </a>
                        )}
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
            )
          ) : (
            <div className="panel">
              <h3>Usuarios similares</h3>
              {similarLoading ? (
                <p className="muted">Cargando usuarios similares...</p>
              ) : similarUsers && similarUsers.length ? (
                <div style={{ overflowX: "auto" }}>
                  <table className="table" style={{ width: "100%", borderCollapse: "collapse" }}>
                    <thead>
                      <tr>
                        <th style={{ textAlign: "left", padding: 8 }}>UserID</th>
                        <th style={{ textAlign: "right", padding: 8 }}>Similarity</th>
                      </tr>
                    </thead>
                    <tbody>
                      {similarUsers.map(s => (
                        <tr key={s.UserID}>
                          <td style={{ padding: 8 }}>{s.UserID}</td>
                          <td style={{ padding: 8, textAlign: "right" }}>{s.Similarity.toFixed(6)}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              ) : (
                <p className="muted">No se encontraron usuarios similares.</p>
              )}
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
