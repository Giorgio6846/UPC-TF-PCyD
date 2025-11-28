import React, { useState } from "react";
import { useNavigate } from "react-router-dom";
import { login as loginApi } from "../api/client";
import { useAuth } from "../context/AuthContext";

const LoginPage: React.FC = () => {
  const [email, setEmail] = useState("test@example.com");
  const [password, setPassword] = useState("123456");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const { login } = useAuth();
  const navigate = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setLoading(true);

    try {
      const res = await loginApi(email, password);
      login(email, res.token);
      navigate("/user");
    } catch (err: any) {
      setError(err?.message ?? "No pudimos iniciar sesion.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <section className="page login-page">
      <div className="page-grid">
        <div className="hero-card">
          <p className="pill">Recomendador de peliculas</p>
          <h1>Descubre el siguiente titulo que te va a encantar.</h1>
          <p className="muted">
            Conecta tu cuenta y explora recomendaciones personalizadas alimentadas por MovieLens y nuestro motor distribuido.
          </p>
          <div className="hero-badges">
            <span className="badge">Insights rapidos</span>
            <span className="badge">Metricas en vivo</span>
            <span className="badge">Token seguro</span>
          </div>
          <div className="hero-bg" aria-hidden />
        </div>

        <div className="panel form-panel">
          <div className="panel-head">
            <p className="pill soft">Acceso seguro</p>
            <h2>Inicia sesion</h2>
            <p className="muted">Usa las credenciales configuradas en tu API.</p>
          </div>

          <form className="form-stack" onSubmit={handleSubmit}>
            <label className="input-group">
              <span>Correo</span>
              <input
                type="email"
                value={email}
                onChange={e => setEmail(e.target.value)}
                placeholder="tu_correo@dominio.com"
                required
              />
            </label>
            <label className="input-group">
              <span>Contrasena</span>
              <input
                type="password"
                value={password}
                onChange={e => setPassword(e.target.value)}
                placeholder="********"
                required
              />
            </label>
            {error && <div className="error-banner">{error}</div>}
            <button type="submit" className="btn primary" disabled={loading}>
              {loading ? "Ingresando..." : "Entrar al panel"}
            </button>
            <p className="micro">
              Demo: <code>test@example.com</code> / <code>123456</code>
            </p>
          </form>
        </div>
      </div>
    </section>
  );
};

export default LoginPage;
