"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useState } from "react";

const navigation = [
  { href: "/seller", label: "Магазин" },
  { href: "/seller/products", label: "Товары" },
  { href: "/seller/inventory", label: "Склад" },
  { href: "/seller/orders", label: "Заказы" },
  { href: "/seller/analytics", label: "Аналитика" },
  { href: "/seller/promotions", label: "Продвижение" },
  { href: "/seller/assistant", label: "ИИ-помощник" },
];

export function SellerNavigation() {
  const pathname = usePathname();
  const [isOpen, setIsOpen] = useState(false);

  return (
    <aside className="border-b border-gray-300 pb-4 md:border-r md:border-b-0 md:pr-4 md:pb-0">
      <button
        type="button"
        aria-expanded={isOpen}
        aria-controls="seller-navigation"
        onClick={() => setIsOpen(!isOpen)}
        className="flex min-h-11 items-center gap-4 border border-gray-300 px-3 text-sm md:hidden"
      >
        Разделы кабинета
        <span aria-hidden="true">{isOpen ? "−" : "+"}</span>
      </button>

      <nav
        id="seller-navigation"
        aria-label="Разделы кабинета продавца"
        className={`${isOpen ? "block" : "hidden"} mt-2 md:mt-0 md:block`}
      >
        <ul>
          {navigation.map(({ href, label }) => {
            const isActive = pathname === href ||
              (href !== "/seller" && pathname.startsWith(`${href}/`));

            return (
              <li key={href}>
                <Link
                  href={href}
                  aria-current={isActive ? "page" : undefined}
                  onClick={() => setIsOpen(false)}
                  className={`flex min-h-11 items-center text-sm hover:underline focus-visible:outline-2 focus-visible:outline-offset-2 ${
                    isActive
                      ? "font-semibold underline"
                      : ""
                  }`}
                >
                  {label}
                </Link>
              </li>
            );
          })}
        </ul>
      </nav>
    </aside>
  );
}
