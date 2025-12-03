import React, { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { useAuth } from "../context/AuthContext";

const Navbar: React.FC = () => {
  const { email, displayName, logout, token } = useAuth();
  const [hidden, setHidden] = useState(false);
  const lastYRef = useRef(0);

  useEffect(() => {
    const onScroll = () => {
      const y = window.scrollY || 0;
      const delta = y - lastYRef.current;
      // Hide when scrolling down past a small threshold; show on scroll up.
      if (delta > 6 && y > 40) {
        setHidden(true);
      } else if (delta < -6) {
        setHidden(false);
      }
      lastYRef.current = y;
    };
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => window.removeEventListener("scroll", onScroll);
  }, []);

  return (
    <header className={`topbar${hidden ? " hidden" : ""}`}>
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
