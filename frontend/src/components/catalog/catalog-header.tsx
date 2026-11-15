"use client";

import Link from "next/link";
import { useDemoStore } from "@/lib/demo-store";

export function CatalogHeader() {
  const { profile } = useDemoStore();

  return (
    <header className="border-b">
      <nav aria-label="Навигация витрины" className="mx-auto flex max-w-6xl flex-wrap items-center justify-between gap-3 px-6 py-4">
        <Link href="/catalog" className="min-w-0 break-words font-semibold">{profile.name}</Link>
        <Link href="/seller" className="text-sm underline">Кабинет продавца</Link>
      </nav>
    </header>
  );
}
