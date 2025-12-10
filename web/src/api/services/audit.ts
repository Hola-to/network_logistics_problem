import { typedClient } from "../typedClient";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";

export const auditService = {
  getLogs: (params?: {
    startTime?: Date;
    endTime?: Date;
    services?: string[];
    actions?: string[];
    userId?: string;
    resourceType?: string;
    limit?: number;
    offset?: number;
  }) =>
    typedClient.getAuditLogs({
      startTime: params?.startTime
        ? timestampFromDate(params.startTime)
        : undefined,
      endTime: params?.endTime ? timestampFromDate(params.endTime) : undefined,
      services: params?.services ?? [],
      actions: params?.actions ?? [],
      userId: params?.userId ?? "",
      resourceType: params?.resourceType ?? "",
      limit: params?.limit ?? 50,
      offset: params?.offset ?? 0,
    }),

  getUserActivity: (params?: {
    userId?: string;
    startTime?: Date;
    endTime?: Date;
    limit?: number;
    offset?: number;
  }) =>
    typedClient.getUserActivity({
      userId: params?.userId ?? "",
      startTime: params?.startTime
        ? timestampFromDate(params.startTime)
        : undefined,
      endTime: params?.endTime ? timestampFromDate(params.endTime) : undefined,
      limit: params?.limit ?? 50,
      offset: params?.offset ?? 0,
    }),

  getStats: (params?: {
    startTime?: Date;
    endTime?: Date;
    groupBy?: "hour" | "day";
  }) =>
    typedClient.getAuditStats({
      startTime: params?.startTime
        ? timestampFromDate(params.startTime)
        : undefined,
      endTime: params?.endTime ? timestampFromDate(params.endTime) : undefined,
      groupBy: params?.groupBy ?? "day",
    }),
};
