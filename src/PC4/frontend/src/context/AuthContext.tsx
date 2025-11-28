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
}

interface AuthContextProps extends AuthState {
  login: (email: string, token: string) => void;
  logout: () => void;
}

const AuthContext = createContext<AuthContextProps | undefined>(undefined);

export const AuthProvider = ({ children }: { children: ReactNode }) => {
  const [auth, setAuth] = useState<AuthState>({
    token: null,
    email: null,
  });

  // cargar de localStorage al refrescar
  useEffect(() => {
    const saved = localStorage.getItem("auth");
    if (saved) {
      const data: { token: string; email: string } = JSON.parse(saved);
      setAuth({ token: data.token, email: data.email });
    }
  }, []);

  const handleLogin = (email: string, token: string) => {
    setAuth({ token, email });
    localStorage.setItem("auth", JSON.stringify({ email, token }));
  };

  const handleLogout = () => {
    setAuth({ token: null, email: null });
    localStorage.removeItem("auth");
  };

  return (
    <AuthContext.Provider
      value={{
        token: auth.token,
        email: auth.email,
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
