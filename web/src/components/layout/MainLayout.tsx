import { Outlet, NavLink } from "react-router-dom";
import { Fragment, useState } from "react";
import { Menu, Transition } from "@headlessui/react";
import {
  HomeIcon,
  MapIcon,
  PlayIcon,
  ChartBarIcon,
  ClockIcon,
  DocumentTextIcon,
  Cog6ToothIcon,
  ArrowRightOnRectangleIcon,
  Bars3Icon,
  XMarkIcon,
} from "@heroicons/react/24/outline";
import { useAuth } from "@/hooks/useAuth";
import clsx from "clsx";

const navigation = [
  { name: "Дашборд", href: "/dashboard", icon: HomeIcon },
  { name: "Редактор сети", href: "/network", icon: MapIcon },
  { name: "Симуляция", href: "/simulation", icon: PlayIcon },
  { name: "Аналитика", href: "/analytics", icon: ChartBarIcon },
  { name: "История", href: "/history", icon: ClockIcon },
  { name: "Отчёты", href: "/reports", icon: DocumentTextIcon },
];

export default function MainLayout() {
  const { user, isAdmin, logout } = useAuth();
  const [sidebarOpen, setSidebarOpen] = useState(false);

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Mobile sidebar backdrop */}
      <Transition show={sidebarOpen} as={Fragment}>
        <div className="lg:hidden">
          <Transition.Child
            as={Fragment}
            enter="transition-opacity ease-linear duration-300"
            enterFrom="opacity-0"
            enterTo="opacity-100"
            leave="transition-opacity ease-linear duration-300"
            leaveFrom="opacity-100"
            leaveTo="opacity-0"
          >
            <div
              className="fixed inset-0 bg-gray-600/75 z-40"
              onClick={() => setSidebarOpen(false)}
            />
          </Transition.Child>
        </div>
      </Transition>

      {/* Sidebar */}
      <aside
        className={clsx(
          "fixed inset-y-0 left-0 z-50 w-64 bg-white border-r border-gray-200 transform transition-transform duration-300 lg:translate-x-0",
          sidebarOpen ? "translate-x-0" : "-translate-x-full",
        )}
      >
        <div className="flex flex-col h-full">
          {/* Logo */}
          <div className="flex items-center justify-between h-16 px-6 border-b border-gray-200">
            <span className="text-xl font-bold text-primary-600">
              Logistics
            </span>
            <button
              className="lg:hidden p-1 rounded-md hover:bg-gray-100"
              onClick={() => setSidebarOpen(false)}
            >
              <XMarkIcon className="w-6 h-6" />
            </button>
          </div>

          {/* Navigation */}
          <nav className="flex-1 px-4 py-4 space-y-1 overflow-y-auto">
            {navigation.map((item) => (
              <NavLink
                key={item.name}
                to={item.href}
                onClick={() => setSidebarOpen(false)}
                className={({ isActive }) =>
                  clsx(
                    "flex items-center px-3 py-2.5 text-sm font-medium rounded-lg transition-colors",
                    isActive
                      ? "bg-primary-50 text-primary-700"
                      : "text-gray-700 hover:bg-gray-100",
                  )
                }
              >
                <item.icon className="w-5 h-5 mr-3 shrink-0" />
                {item.name}
              </NavLink>
            ))}

            {isAdmin && (
              <>
                <div className="pt-4 mt-4 border-t border-gray-200">
                  <p className="px-3 mb-2 text-xs font-semibold text-gray-400 uppercase tracking-wider">
                    Администрирование
                  </p>
                </div>
                <NavLink
                  to="/admin"
                  onClick={() => setSidebarOpen(false)}
                  className={({ isActive }) =>
                    clsx(
                      "flex items-center px-3 py-2.5 text-sm font-medium rounded-lg transition-colors",
                      isActive
                        ? "bg-primary-50 text-primary-700"
                        : "text-gray-700 hover:bg-gray-100",
                    )
                  }
                >
                  <Cog6ToothIcon className="w-5 h-5 mr-3 shrink-0" />
                  Админ-панель
                </NavLink>
              </>
            )}
          </nav>

          {/* User menu */}
          <div className="p-4 border-t border-gray-200">
            <Menu as="div" className="relative">
              <Menu.Button className="flex items-center w-full p-2 text-sm rounded-lg hover:bg-gray-100">
                <div className="w-9 h-9 rounded-full bg-primary-100 flex items-center justify-center">
                  <span className="text-primary-600 font-medium">
                    {user?.username?.charAt(0).toUpperCase() || "U"}
                  </span>
                </div>
                <div className="ml-3 text-left flex-1 min-w-0">
                  <p className="font-medium text-gray-900 truncate">
                    {user?.username}
                  </p>
                  <p className="text-xs text-gray-500 truncate">{user?.role}</p>
                </div>
              </Menu.Button>

              <Transition
                as={Fragment}
                enter="transition ease-out duration-100"
                enterFrom="transform opacity-0 scale-95"
                enterTo="transform opacity-100 scale-100"
                leave="transition ease-in duration-75"
                leaveFrom="transform opacity-100 scale-100"
                leaveTo="transform opacity-0 scale-95"
              >
                <Menu.Items className="absolute bottom-full left-0 right-0 mb-2 bg-white rounded-lg shadow-lg ring-1 ring-black ring-opacity-5 py-1 focus:outline-none">
                  <Menu.Item>
                    {({ active }) => (
                      <button
                        onClick={logout}
                        className={clsx(
                          "flex items-center w-full px-4 py-2 text-sm",
                          active ? "bg-gray-100" : "",
                        )}
                      >
                        <ArrowRightOnRectangleIcon className="w-5 h-5 mr-2" />
                        Выйти
                      </button>
                    )}
                  </Menu.Item>
                </Menu.Items>
              </Transition>
            </Menu>
          </div>
        </div>
      </aside>

      {/* Main content */}
      <div className="lg:pl-64">
        {/* Top bar (mobile) */}
        <div className="sticky top-0 z-10 flex items-center gap-4 px-4 h-16 bg-white border-b border-gray-200 lg:hidden">
          <button
            onClick={() => setSidebarOpen(true)}
            className="p-2 rounded-lg hover:bg-gray-100"
          >
            <Bars3Icon className="w-6 h-6" />
          </button>
          <span className="text-lg font-semibold text-primary-600">
            Logistics
          </span>
        </div>

        {/* Page content */}
        <main className="p-6 lg:p-8">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
