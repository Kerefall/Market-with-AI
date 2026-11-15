"use client";

import { useState, type FormEvent } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { saveStoreProfile, useDemoStore } from "@/lib/demo-store";

export function StoreProfileForm() {
  const { profile, ready } = useDemoStore();
  const [message, setMessage] = useState<{ text: string; error?: boolean }>();

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const data = new FormData(event.currentTarget);
    const name = String(data.get("name") ?? "").trim();
    if (!name) {
      setMessage({ text: "Введите название магазина.", error: true });
      return;
    }
    try {
      saveStoreProfile({ name, description: String(data.get("description") ?? "").trim() });
      setMessage({ text: "Профиль сохранён." });
    } catch {
      setMessage({ text: "Не удалось сохранить профиль в браузере.", error: true });
    }
  }

  if (!ready) return <p role="status">Загрузка профиля…</p>;

  return (
    <form onSubmit={submit} onChange={() => setMessage(undefined)} className="max-w-md space-y-4">
      <label className="block space-y-1 text-sm">
        <span>Название магазина</span>
        <Input name="name" defaultValue={profile.name} required maxLength={100} />
      </label>
      <label className="block space-y-1 text-sm">
        <span>Описание магазина</span>
        <Textarea name="description" defaultValue={profile.description} maxLength={1000} />
      </label>
      <Button type="submit">Сохранить</Button>
      {message && (
        <p role={message.error ? "alert" : "status"} className={message.error ? "text-sm text-destructive" : "text-sm"}>
          {message.text}
        </p>
      )}
    </form>
  );
}
