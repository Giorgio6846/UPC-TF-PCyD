const BASE_URL = import.meta.env.VITE_API_URL || "http://localhost:8080";
const TMDB_KEY = import.meta.env.VITE_TMDB_KEY;
const TMDB_BEARER = import.meta.env.VITE_TMDB_BEARER;
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

export interface MeResponse {
  userId: number;
  email: string;
  name: string;
  lastName: string;
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
  userId: number,
  chunks?: number
): Promise<ResponseMovieJSON> {
  const params = new URLSearchParams();
  params.set("userId", String(userId));
  if (typeof chunks === "number") params.set("chunks", String(chunks));
  const res = await fetch(`${BASE_URL}/api/similarMovies?${params.toString()}`, {
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

export async function fetchMe(token: string): Promise<MeResponse> {
  const res = await fetch(`${BASE_URL}/me`, {
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!res.ok) {
    const txt = await res.text().catch(() => "");
    throw new Error(txt || "No se pudo obtener la información del usuario");
  }

  return res.json();
}

export interface ResourceResponse {
  cpu_percent: number;
  memory_total_kb: number;
  memory_available_kb: number;
  memory_used_kb: number;
  memory_used_percent: number;
  network_bytes_sent: number;
  network_bytes_recv: number;
  workers: string[];
  timestamp_ms: number;
}

export async function fetchResources(token: string): Promise<ResourceResponse> {
  const res = await fetch(`${BASE_URL}/resource`, {
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!res.ok) {
    const txt = await res.text().catch(() => "");
    throw new Error(txt || "No se pudieron obtener los recursos del servidor");
  }

  return res.json();
}

export interface MyMovieEntry {
  Movie: {
    ID: number;
    IMDB?: number;
    TMDB?: number;
    Title: string;
    Genres: string[];
  };
  Rating: number;
}

export interface MyMoviesResponse {
  userId: number;
  moviesRatings: MyMovieEntry[];
}

export async function fetchMyMovies(token: string): Promise<MyMoviesResponse> {
  const res = await fetch(`${BASE_URL}/me/movies`, {
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!res.ok) {
    const txt = await res.text().catch(() => "");
    throw new Error(txt || "No se pudieron obtener las peliculas del usuario");
  }

  return res.json();
}

export interface SimilarUser {
  UserID: number;
  Similarity: number;
}

export interface SimilarUsersResponse {
  similarity: SimilarUser[];
  durationDB: number;
  durationAlgo: number;
}

export async function fetchSimilarUsers(token: string, userId: number): Promise<SimilarUsersResponse> {
  const res = await fetch(`${BASE_URL}/api/similarUsers?userId=${userId}`, {
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!res.ok) {
    const txt = await res.text().catch(() => "");
    throw new Error(txt || "No se pudieron obtener los usuarios similares");
  }

  return res.json();
}

const buildImageUrl = (path: string | undefined, size: string) =>
  path ? `${TMDB_IMG_BASE}/${size}${path}` : null;

const pickImagePath = (items: any[] | undefined, preferredLangs: Array<string | null>) => {
  if (!items?.length) return null;
  // Prefer common languages, otherwise fall back to the first available item.
  for (const lang of preferredLangs) {
    const match = items.find(
      (img: any) =>
        (lang === null && (img?.iso_639_1 === null || img?.iso_639_1 === "")) ||
        img?.iso_639_1 === lang
    );
    if (match?.file_path) return match.file_path;
  }
  return items[0]?.file_path ?? null;
};

const buildTmdbHeaders = () =>
  TMDB_BEARER
    ? {
        Authorization: `Bearer ${TMDB_BEARER}`,
        accept: "application/json"
      }
    : undefined;

const addApiKeyIfNeeded = (params: URLSearchParams) => {
  if (!TMDB_BEARER && TMDB_KEY) params.set("api_key", TMDB_KEY);
};

const formatImdbId = (imdb?: number) => {
  if (!imdb) return null;
  const str = imdb.toString();
  if (str.startsWith("tt")) return str;
  // TMDB espera el prefijo tt y al menos 7 dígitos.
  return `tt${str.padStart(7, "0")}`;
};

export async function fetchTmdbImages(tmdbId?: number, imdbId?: number): Promise<MovieImages> {
  if (!tmdbId && !imdbId) return { posterUrl: null, backdropUrl: null };
  if (!TMDB_KEY && !TMDB_BEARER) {
    console.warn("TMDB key/bearer no configurado; no se buscaran imagenes.");
    return { posterUrl: null, backdropUrl: null };
  }

  const preferredLangs = ["es", "en", null, "ca", "pt", "fr"];

  const tryByTmdbId = async () => {
    if (!tmdbId) return { posterPath: null as string | null, backdropPath: null as string | null, notFound: false };
    const params = new URLSearchParams();
    addApiKeyIfNeeded(params);
    params.set("include_image_language", "*");
    const url = `${TMDB_BASE}/movie/${tmdbId}/images?${params.toString()}`;
    const headers = buildTmdbHeaders();
    const res = await fetch(url, { headers });
    if (res.status === 404) {
      console.warn("TMDB images 404 para tmdbId", tmdbId);
      return { posterPath: null, backdropPath: null, notFound: true };
    }
    if (!res.ok) {
      const body = await res.text().catch(() => "");
      console.error("TMDB images request failed", res.status, res.statusText, body);
      return { posterPath: null, backdropPath: null, notFound: false };
    }
    const data = await res.json();
    return {
      posterPath: pickImagePath(data?.posters, preferredLangs),
      backdropPath: pickImagePath(data?.backdrops, preferredLangs),
      notFound: false
    };
  };

  const tryByImdbId = async () => {
    const imdbFormatted = formatImdbId(imdbId);
    if (!imdbFormatted) return { posterPath: null as string | null, backdropPath: null as string | null };
    const params = new URLSearchParams();
    addApiKeyIfNeeded(params);
    params.set("external_source", "imdb_id");
    const url = `${TMDB_BASE}/find/${imdbFormatted}?${params.toString()}`;
    const headers = buildTmdbHeaders();
    const res = await fetch(url, { headers });
    if (!res.ok) {
      const body = await res.text().catch(() => "");
      console.error("TMDB find request failed", res.status, res.statusText, body);
      return { posterPath: null, backdropPath: null };
    }
    const data = await res.json();
    const movie = data?.movie_results?.[0];
    return {
      posterPath: movie?.poster_path ?? null,
      backdropPath: movie?.backdrop_path ?? null
    };
  };

  try {
    // 1) Intentar con tmdbId
    const byTmdb = await tryByTmdbId();
    let posterPath = byTmdb.posterPath;
    let backdropPath = byTmdb.backdropPath;

    // 2) Si 404 o sin imágenes, intentar con imdbId
    if ((byTmdb.notFound || (!posterPath && !backdropPath)) && imdbId) {
      const byImdb = await tryByImdbId();
      posterPath = posterPath || byImdb.posterPath;
      backdropPath = backdropPath || byImdb.backdropPath;
    }

    return {
      posterUrl: buildImageUrl(posterPath || undefined, "w500"),
      backdropUrl: buildImageUrl(backdropPath || undefined, "w780")
    };
  } catch (err) {
    console.error("TMDB images fetch failed", err);
    return { posterUrl: null, backdropUrl: null };
  }
}
