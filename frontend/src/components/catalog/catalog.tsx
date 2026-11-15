"use client";

import Link from "next/link";
import { useState } from "react";
import { ProductPhoto } from "@/components/catalog/product-photo";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { categories, formatPrice, useDemoStore } from "@/lib/demo-store";

export function Catalog() {
  const { products } = useDemoStore();
  const [search, setSearch] = useState("");
  const [category, setCategory] = useState("");
  const query = search.trim().toLocaleLowerCase("ru-RU");
  const visible = products.filter((product) =>
    product.status === "published" &&
    (!category || product.category === category) &&
    (product.name + " " + product.tags).toLocaleLowerCase("ru-RU").includes(query)
  );

  return (
    <div className="space-y-5">
      <h1 className="text-2xl font-semibold">Каталог</h1>
      <label className="block max-w-sm space-y-1 text-sm">
        <span>Поиск товаров</span>
        <Input type="search" value={search} onChange={(event) => setSearch(event.target.value)} placeholder="Название или тег" />
      </label>
      <div role="group" aria-label="Категории" className="flex flex-wrap gap-2">
        {["", ...categories].map((item) => (
          <Button
            key={item}
            type="button"
            variant={category === item ? "default" : "outline"}
            aria-pressed={category === item}
            onClick={() => setCategory(item)}
          >
            {item || "Все"}
          </Button>
        ))}
      </div>
      <p role="status" className="text-sm text-muted-foreground">
        {visible.length ? "Найдено товаров: " + visible.length : "Товары не найдены."}
      </p>
      <ul className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {visible.map((product) => (
          <li key={product.id} className="min-w-0">
            <Link
              href={"/catalog/" + product.id}
              className="block h-full space-y-2 rounded-lg border p-3 hover:bg-muted/30 focus-visible:outline-2 focus-visible:outline-offset-2"
            >
              <ProductPhoto photo={product.photo} name="" />
              <h2 className="break-words font-medium">{product.name}</h2>
              <p className="font-semibold">{formatPrice(product.price)}</p>
            </Link>
          </li>
        ))}
      </ul>
    </div>
  );
}
