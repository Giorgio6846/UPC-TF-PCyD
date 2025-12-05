import React from "react";
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import LoginPage from "./pages/LoginPage";
import RegisterPage from "./pages/RegisterPage";
import UserDashboard from "./pages/UserDashboard";
import ResourcesPage from "./pages/ResourcesPage";
import Navbar from "./components/Navbar";
import { useAuth } from "./context/AuthContext";

const RequireAuth: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { token } = useAuth();
  if (!token) return <Navigate to="/login" replace />;
  return <>{children}</>;
};

function App() {
  const { token } = useAuth();

  return (
    <BrowserRouter>
      <div className="app-shell">
        <Navbar />
        <Routes>
          <Route path="/login" element={token ? <Navigate to="/user" replace /> : <LoginPage />} />
          <Route path="/register" element={token ? <Navigate to="/user" replace /> : <RegisterPage />} />
          <Route
            path="/user"
            element={
              <RequireAuth>
                <UserDashboard />
              </RequireAuth>
            }
          />
          <Route
            path="/resources"
            element={
              <RequireAuth>
                <ResourcesPage />
              </RequireAuth>
            }
          />
          <Route path="/" element={<Navigate to={token ? "/user" : "/login"} replace />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </div>
    </BrowserRouter>
  );
}

export default App;
