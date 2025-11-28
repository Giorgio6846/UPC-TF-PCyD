import React from "react";
import { Link } from "react-router-dom";
import { useAuth } from "../context/AuthContext";

const Navbar: React.FC = () => {
  const { email, logout, token } = useAuth();

  return (
    <header className="topbar">
      <Link to="/" className="brand">
        <span className="brand-mark">MR</span>
        <div className="brand-copy">
          <strong>MovieRec</strong>
          <small>Recomendaciones al vuelo</small>
        </div>
      </Link>

      <nav className="topbar-links">
        <Link to="/user">Recomendaciones</Link>
        <Link to="/login">Login</Link>
        <Link to="/register">Registro</Link>
      </nav>

      <div className="topbar-actions">
        {email && <span className="chip">{email}</span>}
        {token ? (
          <button className="ghost-button" onClick={logout}>
            Cerrar sesion
          </button>
        ) : (
          <>
            <Link className="ghost-button" to="/login">
              Iniciar sesion
            </Link>
            <Link className="btn primary" to="/register">
              Crear cuenta
            </Link>
          </>
        )}
      </div>
    </header>
  );
};

export default Navbar;
