import { useAuditStats } from "@/hooks/useAudit";
import Card, { CardHeader } from "@/components/ui/Card";
import Spinner from "@/components/ui/Spinner";
import { subDays } from "date-fns";
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  PieChart,
  Pie,
  Cell,
  Legend,
} from "recharts";
import { format } from "date-fns";
import { ru } from "date-fns/locale";
import type { AuditStatsPoint } from "@gen/logistics/gateway/v1/gateway_pb";
import { timestampDate } from "@bufbuild/protobuf/wkt";

const COLORS = ["#3b82f6", "#10b981", "#f59e0b", "#ef4444", "#8b5cf6"];

export default function AdminDashboard() {
  const { data: stats, isLoading } = useAuditStats({
    startTime: subDays(new Date(), 7),
    endTime: new Date(),
    groupBy: "day",
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Spinner size="lg" />
      </div>
    );
  }

  const serviceData = stats?.byService
    ? Object.entries(stats.byService).map(([name, value]) => ({
        name,
        value: Number(value),
      }))
    : [];

  const actionData = stats?.byAction
    ? Object.entries(stats.byAction).map(([name, value]) => ({
        name,
        value: Number(value),
      }))
    : [];

  const timelineData =
    stats?.timeline?.map((point: AuditStatsPoint) => ({
      date: point.timestamp
        ? format(timestampDate(point.timestamp), "dd.MM", { locale: ru })
        : "",
      success: Number(point.successCount),
      failure: Number(point.failureCount),
      total: Number(point.count),
    })) ?? [];

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-gray-900">Обзор системы</h1>

      {/* Stats cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <Card>
          <p className="text-sm text-gray-500">Всего событий</p>
          <p className="text-3xl font-bold text-gray-900">
            {stats?.totalEvents?.toString() ?? "0"}
          </p>
        </Card>
        <Card>
          <p className="text-sm text-gray-500">Успешных</p>
          <p className="text-3xl font-bold text-green-600">
            {stats?.successfulEvents?.toString() ?? "0"}
          </p>
        </Card>
        <Card>
          <p className="text-sm text-gray-500">Ошибок</p>
          <p className="text-3xl font-bold text-red-600">
            {stats?.failedEvents?.toString() ?? "0"}
          </p>
        </Card>
        <Card>
          <p className="text-sm text-gray-500">Уникальных пользователей</p>
          <p className="text-3xl font-bold text-primary-600">
            {stats?.uniqueUsers?.toString() ?? "0"}
          </p>
        </Card>
      </div>

      {/* Timeline chart */}
      <Card>
        <CardHeader title="Активность за неделю" />
        <div className="h-72">
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={timelineData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="date" />
              <YAxis />
              <Tooltip />
              <Legend />
              <Area
                type="monotone"
                dataKey="success"
                name="Успешные"
                stackId="1"
                stroke="#10b981"
                fill="#10b981"
                fillOpacity={0.6}
              />
              <Area
                type="monotone"
                dataKey="failure"
                name="Ошибки"
                stackId="1"
                stroke="#ef4444"
                fill="#ef4444"
                fillOpacity={0.6}
              />
            </AreaChart>
          </ResponsiveContainer>
        </div>
      </Card>

      {/* Pie charts */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card>
          <CardHeader title="По сервисам" />
          <div className="h-64">
            <ResponsiveContainer width="100%" height="100%">
              <PieChart>
                <Pie
                  data={serviceData}
                  cx="50%"
                  cy="50%"
                  innerRadius={40}
                  outerRadius={80}
                  paddingAngle={2}
                  dataKey="value"
                  label={({ name, percent }) =>
                    `${name} (${((percent ?? 0) * 100).toFixed(0)}%)`
                  }
                  labelLine={false}
                >
                  {serviceData.map((_, index) => (
                    <Cell
                      key={`cell-${index}`}
                      fill={COLORS[index % COLORS.length]}
                    />
                  ))}
                </Pie>
                <Tooltip />
              </PieChart>
            </ResponsiveContainer>
          </div>
        </Card>

        <Card>
          <CardHeader title="По действиям" />
          <div className="h-64">
            <ResponsiveContainer width="100%" height="100%">
              <PieChart>
                <Pie
                  data={actionData}
                  cx="50%"
                  cy="50%"
                  innerRadius={40}
                  outerRadius={80}
                  paddingAngle={2}
                  dataKey="value"
                  label={({ name, percent }) =>
                    `${name} (${((percent ?? 0) * 100).toFixed(0)}%)`
                  }
                  labelLine={false}
                >
                  {actionData.map((_, index) => (
                    <Cell
                      key={`cell-${index}`}
                      fill={COLORS[index % COLORS.length]}
                    />
                  ))}
                </Pie>
                <Tooltip />
              </PieChart>
            </ResponsiveContainer>
          </div>
        </Card>
      </div>
    </div>
  );
}
