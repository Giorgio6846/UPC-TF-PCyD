// src/pages/UserDashboard.tsx
import React, { useState } from "react";
import { fetchRecommendedMovies } from "../api/client";

interface JsonMovieResult {
  rank: number;
  movieId: number;
  title: string;
  genres: string[];
  imdb?: number;
  tmdb?: number;
  score: number;
}

interface ResponseMovieJSON {
  results: JsonMovieResult[];
  durationDB: number;
  durationAlgo: number;
  durationMovieFetch: number;
}

const UserDashboard: React.FC = () => {
  const [token, setToken] = useState("");
  const [userId, setUserId] = useState<number | "">("");
  const [data, setData] = useState<ResponseMovieJSON | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const handleFetch = async () => {
    if (!token) {
      setError("Pega aquí el token obtenido en el login.");
      return;
    }
    if (userId === "") {
      setError("Ingresa un userId de MovieLens.");
      return;
    }
    setLoading(true);
    setError(null);
    setData(null);
    try {
      const res = await fetchRecommendedMovies(token, Number(userId));
      setData(res);
    } catch (err: any) {
      setError(err.message || "Error obteniendo recomendaciones");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{ padding: "40px" }}>
      <h1>Panel de Usuario</h1>

      <div style={{ marginBottom: "10px" }}>
        <label>
          Token JWT:
          <input
            type="text"
            value={token}
            onChange={e => setToken(e.target.value)}
            style={{ width: "100%", marginLeft: "5px" }}
          />
        </label>
      </div>

      <div style={{ marginBottom: "10px" }}>
        <label>
          userId:
          <input
            type="number"
            value={userId}
            onChange={e =>
              setUserId(e.target.value === "" ? "" : Number(e.target.value))
            }
            style={{ marginLeft: "5px" }}
          />
        </label>
        <button onClick={handleFetch} disabled={loading} style={{ marginLeft: "10px" }}>
          {loading ? "Consultando..." : "Obtener recomendaciones"}
        </button>
      </div>

      {error && <p style={{ color: "red" }}>{error}</p>}

      {data && (
        <div>
          <h2>Resultados</h2>
          <p>
            Tiempo DB: {data.durationDB} ms | Algoritmo: {data.durationAlgo} ms |
            Fetch películas: {data.durationMovieFetch} ms
          </p>
          <ul>
            {data.results.map(m => (
              <li key={m.movieId}>
                #{m.rank} - {m.title} (score: {m.score.toFixed(3)})
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
};

export default UserDashboard;
