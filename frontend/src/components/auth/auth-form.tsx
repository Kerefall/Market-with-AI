"use client";

import Link from "next/link";
import { useState, type FormEvent } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

type FieldName = "firstName" | "lastName" | "login" | "password" | "repeatPassword";

type Field = {
  name: FieldName;
  label: string;
  autoComplete: string;
  type?: "password";
  minLength?: number;
  maxLength: number;
  hint?: string;
};

const loginField: Field = {
  name: "login",
  label: "Логин",
  autoComplete: "username",
  minLength: 3,
  maxLength: 50,
  hint: "От 3 до 50 символов, без пробелов.",
};

const passwordField: Field = {
  name: "password",
  label: "Пароль",
  autoComplete: "current-password",
  type: "password",
  maxLength: 128,
};

const loginFields = [loginField, passwordField];

const registrationFields: Field[] = [
  { name: "firstName", label: "Имя", autoComplete: "given-name", maxLength: 50 },
  { name: "lastName", label: "Фамилия", autoComplete: "family-name", maxLength: 50 },
  loginField,
  {
    ...passwordField,
    autoComplete: "new-password",
    minLength: 8,
    hint: "Минимум 8 символов.",
  },
  {
    ...passwordField,
    name: "repeatPassword",
    label: "Повтор пароля",
    autoComplete: "new-password",
  },
];

type FieldErrors = Partial<Record<FieldName, string>>;

export function AuthForm({ mode }: { mode: "login" | "register" }) {
  const isRegistration = mode === "register";
  const fields = isRegistration ? registrationFields : loginFields;
  const [errors, setErrors] = useState<FieldErrors>({});
  const [isValid, setIsValid] = useState(false);

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    const data = new FormData(form);
    const nextErrors: FieldErrors = {};

    for (const field of fields) {
      const rawValue = String(data.get(field.name) ?? "");
      const value = field.type === "password" ? rawValue : rawValue.trim();

      if (!value.trim()) {
        nextErrors[field.name] = "Заполните это поле.";
      } else if (field.minLength && value.length < field.minLength) {
        nextErrors[field.name] = `Введите хотя бы ${field.minLength} символов.`;
      } else if (value.length > field.maxLength) {
        nextErrors[field.name] = `Максимум ${field.maxLength} символов.`;
      }
    }

    const login = String(data.get("login") ?? "").trim();
    if (!nextErrors.login && /\s/.test(login)) {
      nextErrors.login = "Логин должен быть без пробелов.";
    }

    if (
      isRegistration &&
      !nextErrors.password &&
      !nextErrors.repeatPassword &&
      data.get("password") !== data.get("repeatPassword")
    ) {
      nextErrors.repeatPassword = "Пароли не совпадают.";
    }

    setErrors(nextErrors);
    const firstError = fields.find(({ name }) => nextErrors[name]);
    setIsValid(!firstError);

    if (firstError) {
      const input = form.elements.namedItem(firstError.name);
      if (input instanceof HTMLInputElement) input.focus();
    }
  }

  return (
    <main className="mx-auto max-w-sm p-6">
      <Link href="/seller" className="text-sm underline">К кабинету</Link>
      <h1 id="auth-title" className="mt-5 text-xl font-semibold">
        {isRegistration ? "Регистрация" : "Вход"}
      </h1>

      <form aria-labelledby="auth-title" method="post" onSubmit={handleSubmit} noValidate className="mt-4 space-y-4">
        {fields.map((field) => {
          const { name, label, autoComplete, type, maxLength, minLength, hint } = field;
          const error = errors[name];
          const message = error ?? hint;

          return (
            <div key={name}>
              <label htmlFor={name} className="block text-sm">{label}</label>
              <Input
                id={name}
                name={name}
                type={type ?? "text"}
                autoComplete={autoComplete}
                required
                minLength={minLength}
                maxLength={maxLength}
                aria-invalid={Boolean(error)}
                aria-describedby={message ? `${name}-message` : undefined}
                onChange={() => {
                  setErrors((current) => ({ ...current, [name]: undefined }));
                  setIsValid(false);
                }}
                className="mt-1"
              />
              {message && (
                <p id={`${name}-message`} className={error ? "mt-1 text-sm text-destructive" : "mt-1 text-sm text-muted-foreground"}>
                  {message}
                </p>
              )}
            </div>
          );
        })}

        <Button type="submit">
          {isRegistration ? "Зарегистрироваться" : "Войти"}
        </Button>

        <p role="status" className="text-sm text-gray-700">
          {isValid && "Поля заполнены корректно. Это проверка формы, подключение к серверу добавим позже."}
        </p>
      </form>

      <Link href={isRegistration ? "/login" : "/register"} className="mt-4 inline-block text-sm underline">
        {isRegistration ? "Уже есть аккаунт? Войти" : "Нет аккаунта? Зарегистрироваться"}
      </Link>
    </main>
  );
}
