import { useState } from "react";
import { useUserActivity } from "@/hooks/useAudit";
import { format, subDays } from "date-fns";
import { ru } from "date-fns/locale";
import Card, { CardHeader } from "@/components/ui/Card";
import Input from "@/components/ui/Input";
import Badge from "@/components/ui/Badge";
import Spinner from "@/components/ui/Spinner";
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from "recharts";
import type { AuditEntry } from "@gen/logistics/gateway/v1/gateway_pb";
import { timestampDate } from "@bufbuild/protobuf/wkt";

export default function UserActivity() {
  const [userId, setUserId] = useState("");

  const { data, isLoading } = useUserActivity({
    userId: userId || undefined,
    startTime: subDays(new Date(), 30),
    endTime: new Date(),
    limit: 50,
  });

  const actionData = data?.summary?.actionsByType
    ? Object.entries(data.summary.actionsByType).map(([name, value]) => ({
        name,
        value,
      }))
    : [];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900">
          Активность пользователей
        </h1>

        <Input
          placeholder="ID пользователя"
          value={userId}
          onChange={(e) => setUserId(e.target.value)}
          className="w-64"
        />
      </div>

      {isLoading ? (
        <div className="flex justify-center py-8">
          <Spinner size="lg" />
        </div>
      ) : data ? (
        <>
          {/* Summary cards */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
            <Card>
              <p className="text-sm text-gray-500">Всего действий</p>
              <p className="text-3xl font-bold text-gray-900">
                {data.summary?.totalActions?.toString() ?? "0"}
              </p>
            </Card>
            <Card>
              <p className="text-sm text-gray-500">Успешных</p>
              <p className="text-3xl font-bold text-green-600">
                {data.summary?.successfulActions?.toString() ?? "0"}
              </p>
            </Card>
            <Card>
              <p className="text-sm text-gray-500">Ошибок</p>
              <p className="text-3xl font-bold text-red-600">
                {data.summary?.failedActions?.toString() ?? "0"}
              </p>
            </Card>
            <Card>
              <p className="text-sm text-gray-500">Последняя активность</p>
              <p className="text-lg font-medium text-gray-900">
                {data.summary?.lastActivity
                  ? format(
                      timestampDate(data.summary.lastActivity),
                      "dd.MM.yyyy HH:mm",
                      { locale: ru },
                    )
                  : "—"}
              </p>
            </Card>
          </div>

          {/* Actions chart */}
          {actionData.length > 0 && (
            <Card>
              <CardHeader title="По типам действий" />
              <div className="h-64">
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={actionData}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="name" />
                    <YAxis />
                    <Tooltip />
                    <Bar dataKey="value" fill="#3b82f6" />
                  </BarChart>
                </ResponsiveContainer>
              </div>
            </Card>
          )}

          {/* Recent activity */}
          <Card>
            <CardHeader title="Последние действия" />
            <div className="space-y-3">
              {data.entries.slice(0, 10).map((entry: AuditEntry) => (
                <div
                  key={entry.id}
                  className="flex items-center justify-between py-2 border-b border-gray-100 last:border-0"
                >
                  <div className="flex items-center gap-3">
                    <Badge
                      variant={
                        entry.outcome === "SUCCESS" ? "success" : "error"
                      }
                    >
                      {entry.action}
                    </Badge>
                    <span className="text-gray-600">{entry.method}</span>
                  </div>
                  <div className="text-sm text-gray-500">
                    {entry.timestamp
                      ? format(
                          timestampDate(entry.timestamp),
                          "dd.MM.yyyy HH:mm:ss",
                          { locale: ru },
                        )
                      : "—"}
                  </div>
                </div>
              ))}
            </div>
          </Card>
        </>
      ) : (
        <Card className="text-center py-8">
          <p className="text-gray-500">
            Введите ID пользователя для просмотра активности
          </p>
        </Card>
      )}
    </div>
  );
}
