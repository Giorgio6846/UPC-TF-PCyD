import React, { useState } from "react";
import { useNavigate } from "react-router-dom";
import { register as registerApi, login as loginApi } from "../api/client";
import { useAuth } from "../context/AuthContext";

const RegisterPage: React.FC = () => {
  const [userId, setUserId] = useState("3");
  const [email, setEmail] = useState("test@test.com");
  const [password, setPassword] = useState("test1234");
  const [name, setName] = useState("Fabio");
  const [lastName, setLastName] = useState("Mancusi");
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const { login } = useAuth();
  const navigate = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setSuccess(null);

    if (!userId.trim()) {
      setError("UserID es obligatorio");
      return;
    }

    setLoading(true);

    try {
      const payload = {
        UserID: Number(userId),
        Email: email,
        Password: password,
        Name: name,
        LastName: lastName
      };
      const res = await registerApi(payload);
      setSuccess(res.message ?? "Usuario creado. Iniciando sesion...");
      if (res.token) {
        login(email, res.token, `${name} ${lastName}`);
      } else {
        const loginRes = await loginApi(email, password);
        login(email, loginRes.token, `${name} ${lastName}`);
      }
      navigate("/user");
    } catch (err: any) {
      setError(err?.message ?? "No se pudo registrar");
    } finally {
      setLoading(false);
    }
  };

  return (
    <section className="page login-page">
      <div className="page-grid">
        <div className="hero-card">
          <p className="pill">Crea tu cuenta</p>
          <h1>Configura tu usuario y prueba el motor de recomendaciones.</h1>
          <p className="muted"></p>
          <div className="hero-badges" />
          <div className="hero-bg" aria-hidden />
        </div>

        <div className="panel form-panel">
          <div className="panel-head">
            <h2>Registrate</h2>
            <p className="muted">Completa los datos para crear tu cuenta.</p>
          </div>

          <form className="form-stack" onSubmit={handleSubmit}>
            <label className="input-group">
              <span>UserID</span>
              <input
                type="number"
                min="1"
                value={userId}
                onChange={e => setUserId(e.target.value)}
                placeholder="3"
                required
              />
            </label>
            <div className="form-row">
              <label className="input-group inline">
                <span>Nombre</span>
                <input value={name} onChange={e => setName(e.target.value)} placeholder="Fabio" required />
              </label>
              <label className="input-group inline">
                <span>Apellido</span>
                <input value={lastName} onChange={e => setLastName(e.target.value)} placeholder="Mancusi" required />
              </label>
            </div>
            <label className="input-group">
              <span>Correo</span>
              <input
                type="email"
                value={email}
                onChange={e => setEmail(e.target.value)}
                placeholder="test@test.com"
                required
              />
            </label>
            <label className="input-group">
              <span>Contrasena</span>
              <input
                type="password"
                value={password}
                onChange={e => setPassword(e.target.value)}
                placeholder="test1234"
                required
              />
            </label>
            {error && <div className="error-banner">{error}</div>}
            {success && <div className="status-chip ok">{success}</div>}
            <button type="submit" className="btn primary" disabled={loading}>
              {loading ? "Creando..." : "Crear cuenta"}
            </button>
            <p className="micro">Usa el UserID que tengas disponible en tu backend o dataset.</p>
          </form>
        </div>
      </div>
    </section>
  );
};

export default RegisterPage;
