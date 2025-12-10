import { useState, useMemo } from "react";
import { useAuditLogs } from "@/hooks/useAudit";
import Card from "@/components/ui/Card";
import Table from "@/components/ui/Table";
import Select from "@/components/ui/Select";
import Badge from "@/components/ui/Badge";
import { ColumnDef } from "@tanstack/react-table";
import { format } from "date-fns";
import { ru } from "date-fns/locale";
import type { AuditEntry } from "@gen/logistics/gateway/v1/gateway_pb";
import { timestampDate } from "@bufbuild/protobuf/wkt";

const OUTCOME_VARIANTS: Record<
  string,
  "success" | "error" | "warning" | "default"
> = {
  SUCCESS: "success",
  FAILURE: "error",
  DENIED: "warning",
  ERROR: "error",
};

export default function AuditLogs() {
  const [filters, setFilters] = useState({
    services: [] as string[],
    actions: [] as string[],
    limit: 50,
    offset: 0,
  });

  const { data, isLoading } = useAuditLogs(filters);

  const columns = useMemo<ColumnDef<AuditEntry>[]>(
    () => [
      {
        accessorKey: "timestamp",
        header: "Время",
        cell: ({ row }) => {
          const ts = row.original.timestamp;
          return ts
            ? format(timestampDate(ts), "dd.MM.yyyy HH:mm:ss", {
                locale: ru,
              })
            : "—";
        },
      },
      {
        accessorKey: "service",
        header: "Сервис",
      },
      {
        accessorKey: "method",
        header: "Метод",
        cell: ({ getValue }) => {
          const method = getValue<string>();
          const parts = method.split("/");
          return parts[parts.length - 1];
        },
      },
      {
        accessorKey: "action",
        header: "Действие",
      },
      {
        accessorKey: "outcome",
        header: "Результат",
        cell: ({ getValue }) => {
          const outcome = getValue<string>();
          return (
            <Badge variant={OUTCOME_VARIANTS[outcome] ?? "default"}>
              {outcome}
            </Badge>
          );
        },
      },
      {
        accessorKey: "username",
        header: "Пользователь",
        cell: ({ getValue }) => getValue<string>() || "—",
      },
      {
        accessorKey: "clientIp",
        header: "IP",
        cell: ({ getValue }) => getValue<string>() || "—",
      },
      {
        accessorKey: "durationMs",
        header: "Время (мс)",
        cell: ({ getValue }) => `${getValue<number>()} мс`,
      },
      {
        accessorKey: "errorMessage",
        header: "Ошибка",
        cell: ({ getValue }) => {
          const error = getValue<string>();
          if (!error) return "—";
          return (
            <span className="text-red-600 truncate max-w-xs" title={error}>
              {error.slice(0, 50)}...
            </span>
          );
        },
      },
    ],
    [],
  );

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900">Аудит логи</h1>

        <div className="flex gap-4">
          <Select
            value={filters.services[0] ?? ""}
            onChange={(e) =>
              setFilters((prev) => ({
                ...prev,
                services: e.target.value ? [e.target.value] : [],
              }))
            }
            options={[
              { value: "", label: "Все сервисы" },
              { value: "gateway-svc", label: "Gateway" },
              { value: "auth-svc", label: "Auth" },
              { value: "solver-svc", label: "Solver" },
              { value: "analytics-svc", label: "Analytics" },
            ]}
          />

          <Select
            value={filters.actions[0] ?? ""}
            onChange={(e) =>
              setFilters((prev) => ({
                ...prev,
                actions: e.target.value ? [e.target.value] : [],
              }))
            }
            options={[
              { value: "", label: "Все действия" },
              { value: "LOGIN", label: "Login" },
              { value: "LOGOUT", label: "Logout" },
              { value: "SOLVE", label: "Solve" },
              { value: "CREATE", label: "Create" },
              { value: "READ", label: "Read" },
              { value: "UPDATE", label: "Update" },
              { value: "DELETE", label: "Delete" },
            ]}
          />
        </div>
      </div>

      <Card padding="none">
        <Table
          data={data?.entries ?? []}
          columns={columns}
          loading={isLoading}
          pageSize={20}
        />
      </Card>

      {data && (
        <p className="text-sm text-gray-500">
          Показано {data.entries.length} из {data.totalCount?.toString()}{" "}
          записей
        </p>
      )}
    </div>
  );
}
