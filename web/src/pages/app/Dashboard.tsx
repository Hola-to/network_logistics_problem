import { useQuery } from "@tanstack/react-query";
import { solverService, historyService } from "@/api/services";
import Card from "@/components/ui/Card";
import Spinner from "@/components/ui/Spinner";
import {
  CheckCircleIcon,
  ExclamationCircleIcon,
} from "@heroicons/react/24/outline";
import clsx from "clsx";
import { ServiceHealth } from "@gen/logistics/gateway/v1/gateway_pb";

export default function Dashboard() {
  const healthQuery = useQuery({
    queryKey: ["health"],
    queryFn: () => solverService.health(),
    refetchInterval: 30000,
  });

  const statsQuery = useQuery({
    queryKey: ["statistics"],
    queryFn: () => historyService.getStatistics(),
  });

  const health = healthQuery.data;
  const stats = statsQuery.data;

  if (healthQuery.isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Spinner size="lg" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-gray-900">Дашборд</h1>

      {/* Service health */}
      <Card>
        <h2 className="text-lg font-semibold mb-4">Состояние сервисов</h2>
        {health?.services ? (
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            {/* Явно приводим тип rawService к ServiceHealth */}
            {Object.entries(health.services).map(([name, rawService]) => {
              const service = rawService as ServiceHealth;

              return (
                <div
                  key={name}
                  className={clsx(
                    "p-4 rounded-lg border",
                    service.status === "HEALTHY"
                      ? "bg-green-50 border-green-200"
                      : "bg-red-50 border-red-200",
                  )}
                >
                  <div className="flex items-center gap-2">
                    {service.status === "HEALTHY" ? (
                      <CheckCircleIcon className="w-5 h-5 text-green-600" />
                    ) : (
                      <ExclamationCircleIcon className="w-5 h-5 text-red-600" />
                    )}
                    <span className="font-medium capitalize">{name}</span>
                  </div>
                  <p className="text-sm text-gray-500 mt-1">
                    {service.latencyMs}ms
                  </p>
                </div>
              );
            })}
          </div>
        ) : (
          <p className="text-gray-500">Нет данных о сервисах</p>
        )}
      </Card>

      {/* Statistics */}
      {stats && (
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
          <Card>
            <p className="text-sm text-gray-500">Всего расчётов</p>
            <p className="text-3xl font-bold text-gray-900">
              {stats.totalCalculations}
            </p>
          </Card>
          <Card>
            <p className="text-sm text-gray-500">Средний поток</p>
            <p className="text-3xl font-bold text-primary-600">
              {stats.averageMaxFlow.toFixed(2)}
            </p>
          </Card>
          <Card>
            <p className="text-sm text-gray-500">Средняя стоимость</p>
            <p className="text-3xl font-bold text-gray-900">
              ₽{stats.averageCost.toFixed(2)}
            </p>
          </Card>
          <Card>
            <p className="text-sm text-gray-500">Среднее время</p>
            <p className="text-3xl font-bold text-gray-900">
              {stats.averageComputationTimeMs.toFixed(0)} мс
            </p>
          </Card>
        </div>
      )}

      {/* Quick actions */}
      <Card>
        <h2 className="text-lg font-semibold mb-4">Быстрые действия</h2>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <a
            href="/network"
            className="p-4 bg-primary-50 rounded-lg hover:bg-primary-100 transition-colors text-center"
          >
            <p className="font-medium text-primary-700">Создать сеть</p>
            <p className="text-sm text-primary-600">Редактор графа</p>
          </a>
          <a
            href="/simulation"
            className="p-4 bg-green-50 rounded-lg hover:bg-green-100 transition-colors text-center"
          >
            <p className="font-medium text-green-700">Симуляция</p>
            <p className="text-sm text-green-600">What-If анализ</p>
          </a>
          <a
            href="/analytics"
            className="p-4 bg-blue-50 rounded-lg hover:bg-blue-100 transition-colors text-center"
          >
            <p className="font-medium text-blue-700">Аналитика</p>
            <p className="text-sm text-blue-600">Анализ потока</p>
          </a>
          <a
            href="/history"
            className="p-4 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors text-center"
          >
            <p className="font-medium text-gray-700">История</p>
            <p className="text-sm text-gray-600">Прошлые расчёты</p>
          </a>
        </div>
      </Card>
    </div>
  );
}
