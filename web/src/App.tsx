import { Routes, Route, Navigate } from "react-router-dom";

// Layouts
import AuthLayout from "@/components/layout/AuthLayout";
import MainLayout from "@/components/layout/MainLayout";
import AdminLayout from "@/components/layout/AdminLayout";
import ProtectedRoute from "@/components/layout/ProtectedRoute";

// Auth pages
import Login from "@/pages/auth/Login";
import Register from "@/pages/auth/Register";

// App pages
import Dashboard from "@/pages/app/Dashboard";
import NetworkEditor from "@/pages/app/NetworkEditor";
import Simulation from "@/pages/app/Simulation";
import Analytics from "@/pages/app/Analytics";
import History from "@/pages/app/History";
import Reports from "@/pages/app/Reports";

// Admin pages
import AdminDashboard from "@/pages/admin/Dashboard";
import AuditLogs from "@/pages/admin/AuditLogs";
import UserActivity from "@/pages/admin/UserActivity";
import Stats from "@/pages/admin/Stats";

export default function App() {
  return (
    <Routes>
      {/* Public routes */}
      <Route element={<AuthLayout />}>
        <Route path="/login" element={<Login />} />
        <Route path="/register" element={<Register />} />
      </Route>

      {/* Protected app routes */}
      <Route
        element={
          <ProtectedRoute>
            <MainLayout />
          </ProtectedRoute>
        }
      >
        <Route path="/" element={<Navigate to="/dashboard" replace />} />
        <Route path="/dashboard" element={<Dashboard />} />
        <Route path="/network" element={<NetworkEditor />} />
        <Route path="/simulation" element={<Simulation />} />
        <Route path="/analytics" element={<Analytics />} />
        <Route path="/history" element={<History />} />
        <Route path="/reports" element={<Reports />} />
      </Route>

      {/* Admin routes */}
      <Route
        path="/admin"
        element={
          <ProtectedRoute requireAdmin>
            <AdminLayout />
          </ProtectedRoute>
        }
      >
        <Route index element={<AdminDashboard />} />
        <Route path="audit" element={<AuditLogs />} />
        <Route path="activity" element={<UserActivity />} />
        <Route path="stats" element={<Stats />} />
      </Route>

      {/* Fallback */}
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
