import React, { useEffect, useState } from "react";
import { useAuth } from "../context/AuthContext";
import { fetchResources } from "../api/client";

const formatKB = (kb: number) => {
  if (kb >= 1024 * 1024) return `${(kb / (1024 * 1024)).toFixed(2)} GB`;
  if (kb >= 1024) return `${(kb / 1024).toFixed(2)} MB`;
  return `${kb} KB`;
};

const ResourcesPage: React.FC = () => {
  const { token } = useAuth();
  const [data, setData] = useState<any | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!token) return;
    let cancelled = false;
    let inFlight = false;
    const POLL_MS = 500; // poll interval in milliseconds (requested)

    const loadOnce = async () => {
      if (cancelled) return;
      if (inFlight) return; // avoid overlapping requests
      inFlight = true;
      try {
        const res = await fetchResources(token);
        if (cancelled) return;
        setData(res);
        setError(null);
      } catch (err: any) {
        if (cancelled) return;
        setError(err?.message ?? "Error obteniendo recursos");
      } finally {
        inFlight = false;
        setLoading(false);
      }
    };

    // initial load
    setLoading(true);
    loadOnce();

    // start polling
    const id = setInterval(() => {
      void loadOnce();
    }, POLL_MS);

    return () => {
      cancelled = true;
      clearInterval(id);
    };
  }, [token]);

  return (
    <section className="page dashboard">
      <div className="page-header">
        <div>
          <p className="pill">Recursos del servidor</p>
          <h1>Estado</h1>
          <p className="muted">Información del servidor obtenida desde la API.</p>
        </div>
      </div>

      <div className="panel form-panel">
        {loading ? (
          <p className="muted">Cargando estado del servidor...</p>
        ) : error ? (
          <div className="error-banner">{error}</div>
        ) : data ? (
          <div>
            <div className="stat-grid">
              <div className="stat-card">
                <p className="muted">CPU</p>
                <strong>{data.cpu_percent?.toFixed(2)} %</strong>
              </div>
              <div className="stat-card">
                <p className="muted">Memoria total</p>
                <strong>{formatKB(data.memory_total_kb)}</strong>
              </div>
              <div className="stat-card">
                <p className="muted">Memoria usada</p>
                <strong>{formatKB(data.memory_used_kb)} ({data.memory_used_percent?.toFixed(2)}%)</strong>
              </div>
              <div className="stat-card">
                <p className="muted">Network (sent / recv)</p>
                <strong>{data.network_bytes_sent} / {data.network_bytes_recv}</strong>
              </div>
            </div>

            <div style={{ marginTop: 12 }}>
              <h3>Workers ({data.workers?.length ?? 0})</h3>
              <ul>
                {Array.isArray(data.workers) && data.workers.map((w: string) => <li key={w}>{w}</li>)}
              </ul>
              <p className="micro muted">Último muestreo: {data.timestamp_ms ? new Date(data.timestamp_ms).toLocaleString() : "-"}</p>
            </div>
          </div>
        ) : (
          <p className="muted">Ningún dato disponible.</p>
        )}
      </div>
    </section>
  );
};

export default ResourcesPage;
