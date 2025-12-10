import { useState } from "react";
import { useAuditStats } from "@/hooks/useAudit";
import { subDays, subMonths, format } from "date-fns";
import { ru } from "date-fns/locale";
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from "recharts";
import Card, { CardHeader } from "@/components/ui/Card";
import Spinner from "@/components/ui/Spinner";
import type { AuditStatsPoint } from "@gen/logistics/gateway/v1/gateway_pb";
import { timestampDate } from "@bufbuild/protobuf/wkt";

type Period = "7d" | "30d" | "90d";

export default function AuditStats() {
  const [period, setPeriod] = useState<Period>("7d");
  const [groupBy, setGroupBy] = useState<"hour" | "day">("day");

  const getStartDate = (): Date => {
    switch (period) {
      case "7d":
        return subDays(new Date(), 7);
      case "30d":
        return subMonths(new Date(), 1);
      case "90d":
        return subMonths(new Date(), 3);
    }
  };

  const { data, isLoading } = useAuditStats({
    startTime: getStartDate(),
    endTime: new Date(),
    groupBy,
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Spinner size="lg" />
      </div>
    );
  }

  const totalEvents = Number(data?.totalEvents ?? 0n);
  const successfulEvents = Number(data?.successfulEvents ?? 0n);
  const failedEvents = Number(data?.failedEvents ?? 0n);
  const uniqueUsers = Number(data?.uniqueUsers ?? 0n);

  const successRate =
    totalEvents > 0 ? ((successfulEvents / totalEvents) * 100).toFixed(1) : "0";

  const timelineData =
    data?.timeline?.map((point: AuditStatsPoint) => ({
      timestamp: point.timestamp
        ? timestampDate(point.timestamp).toISOString()
        : "",
      count: Number(point.count ?? 0n),
      successCount: Number(point.successCount ?? 0n),
      failureCount: Number(point.failureCount ?? 0n),
    })) ?? [];

  const byService: Record<string, number> = {};
  if (data?.byService) {
    for (const [key, value] of Object.entries(data.byService)) {
      byService[key] = Number(value);
    }
  }

  const byAction: Record<string, number> = {};
  if (data?.byAction) {
    for (const [key, value] of Object.entries(data.byAction)) {
      byAction[key] = Number(value);
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900">Статистика аудита</h1>

        <div className="flex gap-4">
          <select
            className="input w-32"
            value={period}
            onChange={(e) => setPeriod(e.target.value as Period)}
          >
            <option value="7d">7 дней</option>
            <option value="30d">30 дней</option>
            <option value="90d">90 дней</option>
          </select>

          <select
            className="input w-32"
            value={groupBy}
            onChange={(e) => setGroupBy(e.target.value as "hour" | "day")}
          >
            <option value="hour">По часам</option>
            <option value="day">По дням</option>
          </select>
        </div>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-1 md:grid-cols-5 gap-6">
        <Card>
          <p className="text-sm text-gray-500">Всего событий</p>
          <p className="text-3xl font-bold text-gray-900">{totalEvents}</p>
        </Card>
        <Card>
          <p className="text-sm text-gray-500">Успешных</p>
          <p className="text-3xl font-bold text-green-600">
            {successfulEvents}
          </p>
        </Card>
        <Card>
          <p className="text-sm text-gray-500">Ошибок</p>
          <p className="text-3xl font-bold text-red-600">{failedEvents}</p>
        </Card>
        <Card>
          <p className="text-sm text-gray-500">% успеха</p>
          <p className="text-3xl font-bold text-primary-600">{successRate}%</p>
        </Card>
        <Card>
          <p className="text-sm text-gray-500">Уникальных пользователей</p>
          <p className="text-3xl font-bold text-gray-900">{uniqueUsers}</p>
        </Card>
      </div>

      {/* Timeline */}
      <Card>
        <CardHeader title="Timeline" />
        <div className="h-80">
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={timelineData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis
                dataKey="timestamp"
                tickFormatter={(value) => {
                  if (!value) return "";
                  return format(
                    new Date(value),
                    groupBy === "hour" ? "HH:mm" : "dd.MM",
                    { locale: ru },
                  );
                }}
              />
              <YAxis />
              <Tooltip
                labelFormatter={(value) => {
                  if (!value) return "";
                  return format(new Date(value), "dd.MM.yyyy HH:mm", {
                    locale: ru,
                  });
                }}
              />
              <Legend />
              <Line
                type="monotone"
                dataKey="count"
                name="Всего"
                stroke="#6b7280"
                strokeWidth={2}
              />
              <Line
                type="monotone"
                dataKey="successCount"
                name="Успешные"
                stroke="#10b981"
                strokeWidth={2}
              />
              <Line
                type="monotone"
                dataKey="failureCount"
                name="Ошибки"
                stroke="#ef4444"
                strokeWidth={2}
              />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </Card>

      {/* By service and action */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <Card>
          <CardHeader title="По сервисам" />
          <div className="space-y-2">
            {Object.entries(byService)
              .sort(([, a], [, b]) => b - a)
              .map(([service, count]) => (
                <div
                  key={service}
                  className="flex items-center justify-between py-2 border-b border-gray-100 last:border-0"
                >
                  <span className="text-gray-600">{service}</span>
                  <span className="font-medium">{count}</span>
                </div>
              ))}
            {Object.keys(byService).length === 0 && (
              <p className="text-gray-500 text-center py-4">Нет данных</p>
            )}
          </div>
        </Card>

        <Card>
          <CardHeader title="По действиям" />
          <div className="space-y-2">
            {Object.entries(byAction)
              .sort(([, a], [, b]) => b - a)
              .map(([action, count]) => (
                <div
                  key={action}
                  className="flex items-center justify-between py-2 border-b border-gray-100 last:border-0"
                >
                  <span className="text-gray-600">{action}</span>
                  <span className="font-medium">{count}</span>
                </div>
              ))}
            {Object.keys(byAction).length === 0 && (
              <p className="text-gray-500 text-center py-4">Нет данных</p>
            )}
          </div>
        </Card>
      </div>
    </div>
  );
}
