import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Link } from "react-router-dom";
import { useAuth } from "@/hooks/useAuth";
import Button from "@/components/ui/Button";
import Input from "@/components/ui/Input";

const loginSchema = z.object({
  username: z.string().min(1, "Введите имя пользователя"),
  password: z.string().min(1, "Введите пароль"),
});

type LoginForm = z.infer<typeof loginSchema>;

export default function Login() {
  const { login, isLoginLoading } = useAuth();

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<LoginForm>({
    resolver: zodResolver(loginSchema),
  });

  const onSubmit = (data: LoginForm) => {
    login(data);
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
      <div className="text-center mb-6">
        <h2 className="text-2xl font-bold text-gray-900">Вход в систему</h2>
        <p className="mt-1 text-sm text-gray-600">Войдите в свой аккаунт</p>
      </div>

      <div className="space-y-4">
        <Input
          label="Имя пользователя"
          type="text"
          autoComplete="username"
          error={errors.username?.message}
          {...register("username")}
        />

        <Input
          label="Пароль"
          type="password"
          autoComplete="current-password"
          error={errors.password?.message}
          {...register("password")}
        />
      </div>

      <Button type="submit" className="w-full" loading={isLoginLoading}>
        Войти
      </Button>

      <p className="text-center text-sm text-gray-600">
        Нет аккаунта?{" "}
        <Link to="/register" className="link">
          Зарегистрироваться
        </Link>
      </p>
    </form>
  );
}
