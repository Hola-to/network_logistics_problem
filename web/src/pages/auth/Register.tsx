import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Link } from "react-router-dom";
import { useAuth } from "@/hooks/useAuth";
import Button from "@/components/ui/Button";
import Input from "@/components/ui/Input";

const registerSchema = z
  .object({
    username: z.string().min(3, "Минимум 3 символа"),
    email: z.string().email("Неверный формат email"),
    fullName: z.string().optional(),
    password: z.string().min(6, "Минимум 6 символов"),
    confirmPassword: z.string(),
  })
  .refine((data) => data.password === data.confirmPassword, {
    message: "Пароли не совпадают",
    path: ["confirmPassword"],
  });

type RegisterForm = z.infer<typeof registerSchema>;

export default function Register() {
  const { register: registerUser, isRegisterLoading } = useAuth();

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<RegisterForm>({
    resolver: zodResolver(registerSchema),
  });

  const onSubmit = (data: RegisterForm) => {
    registerUser({
      username: data.username,
      email: data.email,
      password: data.password,
      fullName: data.fullName ?? "",
    });
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
      <div className="text-center mb-6">
        <h2 className="text-2xl font-bold text-gray-900">Регистрация</h2>
        <p className="mt-1 text-sm text-gray-600">Создайте новый аккаунт</p>
      </div>

      <div className="space-y-4">
        <Input
          label="Имя пользователя"
          type="text"
          error={errors.username?.message}
          {...register("username")}
        />

        <Input
          label="Email"
          type="email"
          error={errors.email?.message}
          {...register("email")}
        />

        <Input
          label="Полное имя"
          type="text"
          hint="Опционально"
          error={errors.fullName?.message}
          {...register("fullName")}
        />

        <Input
          label="Пароль"
          type="password"
          error={errors.password?.message}
          {...register("password")}
        />

        <Input
          label="Подтвердите пароль"
          type="password"
          error={errors.confirmPassword?.message}
          {...register("confirmPassword")}
        />
      </div>

      <Button type="submit" className="w-full" loading={isRegisterLoading}>
        Зарегистрироваться
      </Button>

      <p className="text-center text-sm text-gray-600">
        Уже есть аккаунт?{" "}
        <Link to="/login" className="link">
          Войти
        </Link>
      </p>
    </form>
  );
}
