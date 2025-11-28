const BASE_URL = import.meta.env.VITE_API_URL || "http://localhost:8080";
const TMDB_KEY = import.meta.env.VITE_TMDB_KEY;
const TMDB_BASE = "https://api.themoviedb.org/3";

export interface JsonMovieResult {
  rank: number;
  movieId: number;
  title: string;
  genres: string[];
  imdb?: number;
  tmdb?: number;
  score: number;
}

export interface ResponseMovieJSON {
  results: JsonMovieResult[];
  durationDB: number;
  durationAlgo: number;
  durationMovieFetch: number;
}

export interface LoginResponse {
  token: string;
}

export interface RegisterPayload {
  UserID: number;
  Email: string;
  Password: string;
  Name: string;
  LastName: string;
}

export interface RegisterResponse {
  token?: string;
  message?: string;
}

export async function login(email: string, password: string): Promise<LoginResponse> {
  const res = await fetch(`${BASE_URL}/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password })
  });

  if (!res.ok) {
    const txt = await res.text();
    throw new Error(txt || "Credenciales invalidas");
  }

  return res.json();
}

export async function register(payload: RegisterPayload): Promise<RegisterResponse> {
  const res = await fetch(`${BASE_URL}/auth/register`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload)
  });

  const contentType = res.headers.get("content-type") || "";

  if (!res.ok) {
    const txt = await res.text();
    throw new Error(txt || "No se pudo registrar");
  }

  if (contentType.includes("application/json")) {
    return res.json();
  }

  const txt = await res.text();
  return { message: txt || "Registro completado" };
}

export async function fetchRecommendedMovies(
  token: string,
  userId: number
): Promise<ResponseMovieJSON> {
  const res = await fetch(`${BASE_URL}/api/similarMovies?userId=${userId}`, {
    headers: {
      Authorization: `Bearer ${token}`
    }
  });

  if (!res.ok) {
    const txt = await res.text();
    throw new Error(txt || "No se pudieron obtener las recomendaciones");
  }

  return res.json();
}

export async function fetchTmdbPoster(tmdbId?: number): Promise<string | null> {
  if (!tmdbId || !TMDB_KEY) return null;
  const url = `${TMDB_BASE}/movie/${tmdbId}?api_key=${TMDB_KEY}&language=en-US`;
  try {
    const res = await fetch(url);
    if (!res.ok) return null;
    const data = await res.json();
    if (!data.poster_path) return null;
    return `https://image.tmdb.org/t/p/w342${data.poster_path}`;
  } catch (err) {
    console.error("TMDB poster fetch failed", err);
    return null;
  }
}
