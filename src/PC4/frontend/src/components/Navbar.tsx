import React from "react";
import { Link } from "react-router-dom";
import { useAuth } from "../context/AuthContext";

const Navbar: React.FC = () => {
  const { email, displayName, logout, token } = useAuth();

  return (
    <header className="topbar">
      <Link to="/" className="brand">
        <span className="brand-mark">MR</span>
        <div className="brand-copy">
          <strong>MovieRec</strong>
        </div>
      </Link>

      <div className="topbar-actions">
        {(displayName || email) && <span className="chip">{displayName ?? email}</span>}
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
