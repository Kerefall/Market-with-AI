import Link from "next/link";
import type { Metadata } from "next";
import type { ReactNode } from "react";
import { SellerNavigation } from "@/components/seller/seller-navigation";

export const metadata: Metadata = {
  title: {
    default: "Кабинет продавца · Маркетплейс",
    template: "%s · Кабинет продавца",
  },
};

export default function SellerLayout({ children }: { children: ReactNode }) {
  return (
    <div className="min-h-dvh">
      <a
        href="#main-content"
        className="sr-only bg-white p-2 focus:not-sr-only focus:fixed focus:top-2 focus:left-2 focus:z-50"
      >
        Перейти к содержимому
      </a>

      <header className="flex flex-wrap items-center justify-between gap-2 border-b border-gray-300 px-6 py-4">
        <p className="font-medium">Кабинет продавца</p>
        <nav aria-label="Ссылки кабинета" className="flex flex-wrap gap-4 text-sm">
          <Link href="/catalog" className="underline">Витрина</Link>
          <Link href="/login" className="underline">Войти</Link>
        </nav>
      </header>

      <div className="grid gap-6 p-6 md:grid-cols-[180px_minmax(0,1fr)]">
        <SellerNavigation />
        <main id="main-content" tabIndex={-1} className="min-w-0">
          {children}
        </main>
      </div>
    </div>
  );
}
