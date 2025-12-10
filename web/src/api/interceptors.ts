import { Interceptor, ConnectError, Code } from "@connectrpc/connect";
import { useAuthStore } from "@/stores/authStore";
import toast from "react-hot-toast";

/**
 * Auth interceptor - добавляет токен к каждому запросу
 */
export const authInterceptor: Interceptor = (next) => async (req) => {
  const token = useAuthStore.getState().accessToken;

  if (token) {
    req.header.set("Authorization", `Bearer ${token}`);
  }

  return next(req);
};

/**
 * Error interceptor - обрабатывает ошибки
 */
export const errorInterceptor: Interceptor = (next) => async (req) => {
  try {
    return await next(req);
  } catch (error) {
    if (error instanceof ConnectError) {
      switch (error.code) {
        case Code.Unauthenticated:
          // Токен истёк или невалиден
          useAuthStore.getState().logout();
          window.location.href = "/login";
          break;

        case Code.PermissionDenied:
          toast.error("Недостаточно прав");
          break;

        case Code.ResourceExhausted:
          toast.error("Превышен лимит запросов. Подождите немного.");
          break;

        case Code.Unavailable:
          toast.error("Сервис временно недоступен");
          break;

        case Code.InvalidArgument:
          toast.error(error.message || "Неверные данные");
          break;

        default:
          toast.error(error.message || "Произошла ошибка");
      }
    }
    throw error;
  }
};

/**
 * Logging interceptor (только для development)
 */
export const loggingInterceptor: Interceptor = (next) => async (req) => {
  const start = performance.now();
  const method = req.method.name;

  if (import.meta.env.DEV) {
    console.log(`🚀 [API] ${method}`, req.message);
  }

  try {
    const response = await next(req);

    if (import.meta.env.DEV) {
      const duration = (performance.now() - start).toFixed(2);
      console.log(`✅ [API] ${method} (${duration}ms)`, response.message);
    }

    return response;
  } catch (error) {
    if (import.meta.env.DEV) {
      const duration = (performance.now() - start).toFixed(2);
      console.error(`❌ [API] ${method} (${duration}ms)`, error);
    }
    throw error;
  }
};
