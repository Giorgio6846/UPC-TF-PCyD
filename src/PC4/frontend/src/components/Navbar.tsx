import React, { useEffect, useRef, useState } from "react";
import { NavLink } from "react-router-dom";
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
      <NavLink to="/" className="brand">
        <span className="brand-mark">MR</span>
        <div className="brand-copy">
          <strong>MovieRec</strong>
        </div>
      </NavLink>

      <div className="topbar-actions">
        <nav className="topbar-nav">
          <NavLink to="/user" className={({isActive}) => isActive ? 'nav-link active' : 'nav-link'}>Inicio</NavLink>
          {token && (
            <NavLink to="/resources" className={({isActive}) => isActive ? 'nav-link active' : 'nav-link'}>
              Recursos
            </NavLink>
          )}
        </nav>

        {(displayName || email) && <span className="chip">{displayName ?? email}</span>}
        {token ? (
          <button className="ghost-button" onClick={logout}>
            Cerrar sesión
          </button>
        ) : (
          <>
            <NavLink to="/login" className="ghost-button">
              Iniciar sesión
            </NavLink>
            <NavLink to="/register" className="btn primary">
              Crear cuenta
            </NavLink>
          </>
        )}
      </div>
    </header>
  );
};

export default Navbar;
