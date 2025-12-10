import { useQuery } from "@tanstack/react-query";
import { auditService } from "@/api/services";

interface AuditLogsParams {
  startTime?: Date;
  endTime?: Date;
  services?: string[];
  actions?: string[];
  userId?: string;
  resourceType?: string;
  limit?: number;
  offset?: number;
}

interface UserActivityParams {
  userId?: string;
  startTime?: Date;
  endTime?: Date;
  limit?: number;
  offset?: number;
}

interface AuditStatsParams {
  startTime?: Date;
  endTime?: Date;
  groupBy?: "hour" | "day";
}

export function useAuditLogs(params: AuditLogsParams) {
  return useQuery({
    queryKey: ["auditLogs", params],
    queryFn: () => auditService.getLogs(params),
    staleTime: 30 * 1000,
  });
}

export function useUserActivity(params: UserActivityParams) {
  return useQuery({
    queryKey: ["userActivity", params],
    queryFn: () => auditService.getUserActivity(params),
    staleTime: 30 * 1000,
    enabled: !!params.userId,
  });
}

export function useAuditStats(params: AuditStatsParams) {
  return useQuery({
    queryKey: ["auditStats", params],
    queryFn: () => auditService.getStats(params),
    staleTime: 60 * 1000,
  });
}
