import React, {
  createContext,
  useContext,
  useState,
  useEffect,
  ReactNode,
} from "react";
import type { LoginResponse } from "../api/client";

interface AuthState {
  token: string | null;
  email: string | null;
  displayName: string | null;
}

interface AuthContextProps extends AuthState {
  login: (email: string, token: string, displayName?: string | null) => void;
  logout: () => void;
}

const AuthContext = createContext<AuthContextProps | undefined>(undefined);

export const AuthProvider = ({ children }: { children: ReactNode }) => {
  const [auth, setAuth] = useState<AuthState>({
    token: null,
    email: null,
    displayName: null,
  });

  // cargar de localStorage al refrescar
  useEffect(() => {
    const saved = localStorage.getItem("auth");
    if (saved) {
      const data: { token: string; email: string; displayName?: string | null } = JSON.parse(saved);
      setAuth({ token: data.token, email: data.email, displayName: data.displayName ?? null });
    }
  }, []);

  const handleLogin = (email: string, token: string, displayName?: string | null) => {
    const payload = { token, email, displayName: displayName ?? null };
    setAuth(payload);
    localStorage.setItem("auth", JSON.stringify(payload));
  };

  const handleLogout = () => {
    setAuth({ token: null, email: null, displayName: null });
    localStorage.removeItem("auth");
  };

  return (
    <AuthContext.Provider
      value={{
        token: auth.token,
        email: auth.email,
        displayName: auth.displayName,
        login: handleLogin,
        logout: handleLogout,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth debe usarse dentro de AuthProvider");
  }
  return ctx;
};
