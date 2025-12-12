import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { solverService, historyService } from "@/api/services";
import { useAuthStore } from "@/stores/authStore";
import Card from "@/components/ui/Card";
import Button from "@/components/ui/Button";
import Spinner from "@/components/ui/Spinner";
import {
  CheckCircleIcon,
  ExclamationCircleIcon,
  ArrowPathIcon,
  MapIcon,
  PlayIcon,
  ChartBarIcon,
  ClockIcon,
  ExclamationTriangleIcon,
} from "@heroicons/react/24/outline";
import clsx from "clsx";
import type {
  ServiceHealth,
  StatisticsResponse,
  HealthResponse,
} from "@gen/logistics/gateway/v1/gateway_pb";

// ============================================================================
// Компонент статуса сервисов для пользователей
// ============================================================================

interface UserServiceStatusProps {
  health: HealthResponse | undefined;
  isLoading: boolean;
}

function UserServiceStatus({ health, isLoading }: UserServiceStatusProps) {
  if (isLoading) {
    return (
      <Card>
        <div className="flex items-center gap-3">
          <Spinner size="sm" />
          <span className="text-gray-500">Проверка сервисов...</span>
        </div>
      </Card>
    );
  }

  const status = health?.status;
  const isHealthy = status === "HEALTHY";
  const isDegraded = status === "DEGRADED";

  // Считаем проблемные сервисы
  const services = health?.services ? Object.entries(health.services) : [];
  const unhealthyServices = services.filter(
    ([_, s]) => (s as ServiceHealth).status !== "HEALTHY",
  );

  if (isHealthy) {
    return (
      <Card className="bg-green-50 border-green-200">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 bg-green-100 rounded-full flex items-center justify-center">
            <CheckCircleIcon className="w-6 h-6 text-green-600" />
          </div>
          <div>
            <p className="font-medium text-green-800">Все сервисы работают</p>
            <p className="text-sm text-green-600">
              Система полностью функциональна
            </p>
          </div>
        </div>
      </Card>
    );
  }

  if (isDegraded) {
    return (
      <Card className="bg-yellow-50 border-yellow-200">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 bg-yellow-100 rounded-full flex items-center justify-center">
            <ExclamationTriangleIcon className="w-6 h-6 text-yellow-600" />
          </div>
          <div>
            <p className="font-medium text-yellow-800">
              Частичная работоспособность
            </p>
            <p className="text-sm text-yellow-600">
              {unhealthyServices.length} из {services.length} сервисов
              недоступны. Некоторые функции могут быть ограничены.
            </p>
          </div>
        </div>
      </Card>
    );
  }

  return (
    <Card className="bg-red-50 border-red-200">
      <div className="flex items-center gap-3">
        <div className="w-10 h-10 bg-red-100 rounded-full flex items-center justify-center">
          <ExclamationCircleIcon className="w-6 h-6 text-red-600" />
        </div>
        <div>
          <p className="font-medium text-red-800">Сервисы недоступны</p>
          <p className="text-sm text-red-600">
            Возникли проблемы с подключением. Попробуйте обновить страницу.
          </p>
        </div>
      </div>
    </Card>
  );
}

// ============================================================================
// Компонент статуса сервисов для админов
// ============================================================================

interface AdminServiceStatusProps {
  health: HealthResponse | undefined;
  isLoading: boolean;
}

function AdminServiceStatus({ health, isLoading }: AdminServiceStatusProps) {
  if (isLoading) {
    return (
      <Card>
        <div className="flex items-center gap-3">
          <Spinner size="sm" />
          <span className="text-gray-500">Загрузка состояния сервисов...</span>
        </div>
      </Card>
    );
  }

  return (
    <Card>
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-semibold">Состояние сервисов</h2>
        <div
          className={clsx(
            "px-2 py-1 rounded-full text-xs font-medium",
            health?.status === "HEALTHY"
              ? "bg-green-100 text-green-700"
              : health?.status === "DEGRADED"
                ? "bg-yellow-100 text-yellow-700"
                : "bg-red-100 text-red-700",
          )}
        >
          {health?.status || "Unknown"}
        </div>
      </div>

      {health?.services && Object.keys(health.services).length > 0 ? (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          {Object.entries(health.services).map(([name, rawService]) => {
            const service = rawService as ServiceHealth;
            const isHealthy = service.status === "HEALTHY";

            return (
              <div
                key={name}
                className={clsx(
                  "p-4 rounded-lg border transition-colors",
                  isHealthy
                    ? "bg-green-50 border-green-200"
                    : "bg-red-50 border-red-200",
                )}
              >
                <div className="flex items-center gap-2">
                  {isHealthy ? (
                    <CheckCircleIcon className="w-5 h-5 text-green-600" />
                  ) : (
                    <ExclamationCircleIcon className="w-5 h-5 text-red-600" />
                  )}
                  <span className="font-medium capitalize">
                    {name.replace("-svc", "").replace("_", " ")}
                  </span>
                </div>
                <div className="mt-2 text-sm text-gray-500">
                  <p>Latency: {service.latencyMs ?? 0}ms</p>
                  {service.version && <p>v{service.version}</p>}
                  {service.address && (
                    <p className="text-xs truncate" title={service.address}>
                      {service.address}
                    </p>
                  )}
                </div>
                {service.error && (
                  <p
                    className="mt-1 text-xs text-red-600 truncate"
                    title={service.error}
                  >
                    {service.error}
                  </p>
                )}
              </div>
            );
          })}
        </div>
      ) : (
        <p className="text-gray-500">Нет данных о сервисах</p>
      )}
    </Card>
  );
}

// ============================================================================
// Главный компонент Dashboard
// ============================================================================

export default function Dashboard() {
  const queryClient = useQueryClient();
  const { isAdmin, user } = useAuthStore();

  // Health check
  const healthQuery = useQuery({
    queryKey: ["health"],
    queryFn: async () => {
      const response = await solverService.health();
      return response as HealthResponse;
    },
    refetchInterval: 30000,
    staleTime: 10000,
  });

  // Statistics
  const statsQuery = useQuery({
    queryKey: ["statistics"],
    queryFn: async () => {
      const response = await historyService.getStatistics();
      return response as StatisticsResponse;
    },
    staleTime: 0,
    refetchOnMount: "always",
  });

  const handleRefresh = () => {
    queryClient.invalidateQueries({ queryKey: ["health"] });
    queryClient.invalidateQueries({ queryKey: ["statistics"] });
    queryClient.invalidateQueries({ queryKey: ["calculations"] });
  };

  const health = healthQuery.data;
  const stats = statsQuery.data;

  const isLoading = healthQuery.isLoading || statsQuery.isLoading;
  const hasError = healthQuery.error || statsQuery.error;

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Spinner size="lg" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Дашборд</h1>
          {user && (
            <p className="text-gray-500 text-sm">
              Добро пожаловать, {user.fullName || user.username}
            </p>
          )}
        </div>
        <Button
          variant="ghost"
          size="sm"
          onClick={handleRefresh}
          loading={healthQuery.isFetching || statsQuery.isFetching}
        >
          <ArrowPathIcon className="w-4 h-4 mr-1" />
          Обновить
        </Button>
      </div>

      {/* Error state */}
      {hasError && (
        <Card className="bg-red-50 border-red-200">
          <div className="flex items-center gap-3">
            <ExclamationCircleIcon className="w-6 h-6 text-red-500" />
            <div>
              <p className="text-red-800 font-medium">Ошибка загрузки данных</p>
              <p className="text-red-600 text-sm">
                {(healthQuery.error as Error)?.message ||
                  (statsQuery.error as Error)?.message}
              </p>
            </div>
          </div>
          <Button variant="secondary" onClick={handleRefresh} className="mt-4">
            Повторить
          </Button>
        </Card>
      )}

      {/* Service status - разное для админов и пользователей */}
      {isAdmin ? (
        <AdminServiceStatus health={health} isLoading={healthQuery.isLoading} />
      ) : (
        <UserServiceStatus health={health} isLoading={healthQuery.isLoading} />
      )}

      {/* Statistics */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
        <Card>
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-500">Мои расчёты</p>
              <p className="text-3xl font-bold text-gray-900">
                {stats?.totalCalculations ?? 0}
              </p>
            </div>
            <div className="w-12 h-12 bg-primary-100 rounded-xl flex items-center justify-center">
              <ClockIcon className="w-6 h-6 text-primary-600" />
            </div>
          </div>
        </Card>

        <Card>
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-500">Средний поток</p>
              <p className="text-3xl font-bold text-primary-600">
                {stats?.averageMaxFlow?.toFixed(1) ?? 0}
              </p>
            </div>
            <div className="w-12 h-12 bg-blue-100 rounded-xl flex items-center justify-center">
              <ChartBarIcon className="w-6 h-6 text-blue-600" />
            </div>
          </div>
        </Card>

        <Card>
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-500">Средняя стоимость</p>
              <p className="text-3xl font-bold text-gray-900">
                ₽{stats?.averageCost?.toFixed(0) ?? 0}
              </p>
            </div>
            <div className="w-12 h-12 bg-green-100 rounded-xl flex items-center justify-center">
              <span className="text-green-600 text-xl">₽</span>
            </div>
          </div>
        </Card>

        <Card>
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-500">Среднее время</p>
              <p className="text-3xl font-bold text-gray-900">
                {stats?.averageComputationTimeMs?.toFixed(0) ?? 0}
                <span className="text-lg font-normal text-gray-500"> мс</span>
              </p>
            </div>
            <div className="w-12 h-12 bg-orange-100 rounded-xl flex items-center justify-center">
              <PlayIcon className="w-6 h-6 text-orange-600" />
            </div>
          </div>
        </Card>
      </div>

      {/* Algorithm usage - только для админов */}
      {isAdmin &&
        stats?.calculationsByAlgorithm &&
        Object.keys(stats.calculationsByAlgorithm).length > 0 && (
          <Card>
            <h2 className="text-lg font-semibold mb-4">
              Использование алгоритмов
            </h2>
            <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
              {Object.entries(stats.calculationsByAlgorithm).map(
                ([algo, count]) => (
                  <div
                    key={algo}
                    className="text-center p-3 bg-gray-50 rounded-lg"
                  >
                    <p className="text-2xl font-bold text-primary-600">
                      {count}
                    </p>
                    <p className="text-sm text-gray-500">{algo}</p>
                  </div>
                ),
              )}
            </div>
          </Card>
        )}

      {/* Quick actions */}
      <Card>
        <h2 className="text-lg font-semibold mb-4">Быстрые действия</h2>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <Link
            to="/network"
            className="p-4 bg-primary-50 rounded-lg hover:bg-primary-100 transition-colors text-center group"
          >
            <div className="w-12 h-12 bg-primary-100 rounded-xl flex items-center justify-center mx-auto mb-2 group-hover:bg-primary-200 transition-colors">
              <MapIcon className="w-6 h-6 text-primary-600" />
            </div>
            <p className="font-medium text-primary-700">Создать сеть</p>
            <p className="text-sm text-primary-600">Редактор графа</p>
          </Link>

          <Link
            to="/simulation"
            className="p-4 bg-green-50 rounded-lg hover:bg-green-100 transition-colors text-center group"
          >
            <div className="w-12 h-12 bg-green-100 rounded-xl flex items-center justify-center mx-auto mb-2 group-hover:bg-green-200 transition-colors">
              <PlayIcon className="w-6 h-6 text-green-600" />
            </div>
            <p className="font-medium text-green-700">Симуляция</p>
            <p className="text-sm text-green-600">What-If анализ</p>
          </Link>

          <Link
            to="/analytics"
            className="p-4 bg-blue-50 rounded-lg hover:bg-blue-100 transition-colors text-center group"
          >
            <div className="w-12 h-12 bg-blue-100 rounded-xl flex items-center justify-center mx-auto mb-2 group-hover:bg-blue-200 transition-colors">
              <ChartBarIcon className="w-6 h-6 text-blue-600" />
            </div>
            <p className="font-medium text-blue-700">Аналитика</p>
            <p className="text-sm text-blue-600">Анализ потока</p>
          </Link>

          <Link
            to="/history"
            className="p-4 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors text-center group"
          >
            <div className="w-12 h-12 bg-gray-100 rounded-xl flex items-center justify-center mx-auto mb-2 group-hover:bg-gray-200 transition-colors">
              <ClockIcon className="w-6 h-6 text-gray-600" />
            </div>
            <p className="font-medium text-gray-700">История</p>
            <p className="text-sm text-gray-600">Прошлые расчёты</p>
          </Link>
        </div>
      </Card>

      {/* Recent activity - только для админов */}
      {isAdmin && stats?.dailyStats && stats.dailyStats.length > 0 && (
        <Card>
          <h2 className="text-lg font-semibold mb-4">
            Активность за последние дни
          </h2>
          <div className="space-y-2">
            {stats.dailyStats.slice(0, 7).map((day) => (
              <div
                key={day.date}
                className="flex items-center justify-between py-2 border-b border-gray-100 last:border-0"
              >
                <span className="text-gray-600">{day.date}</span>
                <div className="flex items-center gap-4">
                  <span className="text-sm">
                    <span className="font-medium">{day.count}</span>
                    <span className="text-gray-500 ml-1">расчётов</span>
                  </span>
                  <span className="text-sm text-primary-600">
                    Σ Flow: {day.totalFlow?.toFixed(0) ?? 0}
                  </span>
                </div>
              </div>
            ))}
          </div>
        </Card>
      )}

      {/* Empty state if no stats */}
      {(!stats || stats.totalCalculations === 0) && (
        <Card className="text-center py-12 bg-gray-50">
          <div className="text-gray-400 text-5xl mb-4">📊</div>
          <p className="text-gray-600 text-lg">Нет данных для отображения</p>
          <p className="text-sm text-gray-400 mt-2">
            Создайте свой первый граф и запустите оптимизацию
          </p>
          <Link to="/network">
            <Button className="mt-6">Начать работу</Button>
          </Link>
        </Card>
      )}
    </div>
  );
}
