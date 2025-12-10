import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { useNavigate } from "react-router-dom";
import toast from "react-hot-toast";
import { authService } from "@/api/services";
import { useAuthStore } from "@/stores/authStore";
import type {
  AuthResponse,
  UserProfile,
} from "@gen/logistics/gateway/v1/gateway_pb";

interface LoginData {
  username: string;
  password: string;
}

interface RegisterData {
  username: string;
  email: string;
  password: string;
  fullName: string;
}

export function useAuth() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const store = useAuthStore();

  const profileQuery = useQuery({
    queryKey: ["profile"],
    // 2. Fix: Cast the generic response to UserProfile
    queryFn: async () => {
      const response = await authService.getProfile();
      return response as unknown as UserProfile;
    },
    enabled: store.isAuthenticated && !store.user,
    retry: false,
  });

  useEffect(() => {
    if (profileQuery.data && !store.user) {
      store.setUser(profileQuery.data);
    }
  }, [profileQuery.data, store.user, store]);

  const loginMutation = useMutation({
    // 3. Fix: Cast the generic response to AuthResponse
    mutationFn: async (data: LoginData) => {
      const response = await authService.login(data);
      return response as unknown as AuthResponse;
    },
    onSuccess: (response: AuthResponse) => {
      if (response.success && response.accessToken && response.refreshToken) {
        store.setTokens(response.accessToken, response.refreshToken);
        if (response.user) {
          store.setUser(response.user);
        }
        queryClient.invalidateQueries({ queryKey: ["profile"] });
        toast.success("Добро пожаловать!");
        navigate("/dashboard");
      } else {
        toast.error(response.errorMessage || "Ошибка входа");
      }
    },
  });

  const registerMutation = useMutation({
    // 4. Fix: Cast the generic response to AuthResponse
    mutationFn: async (data: RegisterData) => {
      const response = await authService.register(data);
      return response as unknown as AuthResponse;
    },
    onSuccess: (response: AuthResponse) => {
      if (response.success && response.accessToken && response.refreshToken) {
        store.setTokens(response.accessToken, response.refreshToken);
        if (response.user) {
          store.setUser(response.user);
        }
        queryClient.invalidateQueries({ queryKey: ["profile"] });
        toast.success("Регистрация успешна!");
        navigate("/dashboard");
      } else {
        toast.error(response.errorMessage || "Ошибка регистрации");
      }
    },
  });

  const logout = async () => {
    try {
      await authService.logout();
    } catch {
      // Ignore errors
    } finally {
      store.logout();
      queryClient.clear();
      navigate("/login");
      toast.success("Вы вышли из системы");
    }
  };

  return {
    isAuthenticated: store.isAuthenticated,
    isAdmin: store.isAdmin,
    user: store.user,
    isLoading: profileQuery.isLoading,
    login: loginMutation.mutate,
    register: registerMutation.mutate,
    logout,
    isLoginLoading: loginMutation.isPending,
    isRegisterLoading: registerMutation.isPending,
  };
}
