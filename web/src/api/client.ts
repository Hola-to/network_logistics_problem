import { createClient, type Client } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";

// Импорт сгенерированного сервиса
import { GatewayService } from "@gen/logistics/gateway/v1/gateway_pb";

// Импорт interceptors
import {
  authInterceptor,
  errorInterceptor,
  loggingInterceptor,
} from "./interceptors";

// Создаём transport с interceptors
const transport = createConnectTransport({
  baseUrl: import.meta.env.VITE_API_URL || "http://localhost:8080",
  interceptors: [loggingInterceptor, authInterceptor, errorInterceptor],
});

// Создаём типизированный клиент
export const client: Client<typeof GatewayService> = createClient(
  GatewayService,
  transport,
);

// Re-export типов для удобства
export type { Client } from "@connectrpc/connect";
export { GatewayService };
