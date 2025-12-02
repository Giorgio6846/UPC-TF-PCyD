const BASE_URL = import.meta.env.VITE_API_URL || "http://localhost:8080";
const TMDB_KEY = import.meta.env.VITE_TMDB_KEY;
const TMDB_BASE = "https://api.themoviedb.org/3";
const TMDB_IMG_BASE = "https://image.tmdb.org/t/p";

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

export interface MovieImages {
  posterUrl: string | null;
  backdropUrl: string | null;
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

const buildImageUrl = (path: string | undefined, size: string) =>
  path ? `${TMDB_IMG_BASE}/${size}${path}` : null;

export async function fetchTmdbImages(tmdbId?: number): Promise<MovieImages> {
  if (!tmdbId || !TMDB_KEY) return { posterUrl: null, backdropUrl: null };
  const url = `${TMDB_BASE}/movie/${tmdbId}/images?api_key=${TMDB_KEY}&include_image_language=en,null,es`;
  try {
    const res = await fetch(url);
    if (!res.ok) return { posterUrl: null, backdropUrl: null };
    const data = await res.json();
    const posterPath = data?.posters?.[0]?.file_path;
    const backdropPath = data?.backdrops?.[0]?.file_path;
    return {
      posterUrl: buildImageUrl(posterPath, "w500"),
      backdropUrl: buildImageUrl(backdropPath, "w780")
    };
  } catch (err) {
    console.error("TMDB images fetch failed", err);
    return { posterUrl: null, backdropUrl: null };
  }
}
