import { Outlet, NavLink, Link } from "react-router-dom";
import {
  ChartPieIcon,
  DocumentMagnifyingGlassIcon,
  UserGroupIcon,
  ChartBarIcon,
  ArrowLeftIcon,
} from "@heroicons/react/24/outline";
import clsx from "clsx";

const adminNavigation = [
  { name: "Обзор", href: "/admin", icon: ChartPieIcon, end: true },
  {
    name: "Аудит логи",
    href: "/admin/audit",
    icon: DocumentMagnifyingGlassIcon,
  },
  { name: "Активность", href: "/admin/activity", icon: UserGroupIcon },
  { name: "Статистика", href: "/admin/stats", icon: ChartBarIcon },
];

export default function AdminLayout() {
  return (
    <div className="min-h-screen bg-gray-100">
      {/* Header */}
      <header className="bg-white border-b border-gray-200">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center gap-4">
              <Link
                to="/dashboard"
                className="flex items-center text-gray-600 hover:text-gray-900 transition-colors"
              >
                <ArrowLeftIcon className="w-5 h-5 mr-2" />
                <span className="hidden sm:inline">Назад к приложению</span>
              </Link>
              <div className="h-6 w-px bg-gray-200" />
              <h1 className="text-xl font-semibold text-gray-900">
                Админ-консоль
              </h1>
            </div>
          </div>
        </div>
      </header>

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="flex flex-col lg:flex-row gap-8">
          {/* Sidebar */}
          <aside className="w-full lg:w-64 shrink-0">
            <nav className="bg-white rounded-xl shadow-sm border border-gray-200 p-2">
              <div className="space-y-1">
                {adminNavigation.map((item) => (
                  <NavLink
                    key={item.name}
                    to={item.href}
                    end={item.end}
                    className={({ isActive }) =>
                      clsx(
                        "flex items-center px-3 py-2.5 text-sm font-medium rounded-lg transition-colors",
                        isActive
                          ? "bg-primary-50 text-primary-700"
                          : "text-gray-600 hover:bg-gray-50",
                      )
                    }
                  >
                    <item.icon className="w-5 h-5 mr-3 shrink-0" />
                    {item.name}
                  </NavLink>
                ))}
              </div>
            </nav>
          </aside>

          {/* Content */}
          <main className="flex-1 min-w-0">
            <Outlet />
          </main>
        </div>
      </div>
    </div>
  );
}
