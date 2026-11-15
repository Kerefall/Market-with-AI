"use client";

import Link from "next/link";
import { ProductPhoto } from "@/components/catalog/product-photo";
import { formatPrice, useDemoStore } from "@/lib/demo-store";

export function ProductDetails({ id }: { id: string }) {
  const { products, ready } = useDemoStore();
  const product = products.find((item) => item.id === id && item.status === "published");
  if (!ready) return <p role="status">Загрузка товара…</p>;

  return (
    <div className="space-y-4">
      <Link href="/catalog" className="text-sm underline">К каталогу</Link>
      {product ? (
        <div className="grid max-w-4xl gap-6 md:grid-cols-2">
          <ProductPhoto photo={product.photo} name={product.name} />
          <div className="min-w-0 space-y-3 break-words">
            <p className="text-sm text-muted-foreground">{product.category}</p>
            <h1 className="text-2xl font-semibold">{product.name}</h1>
            <p className="text-xl font-semibold">{formatPrice(product.price)}</p>
            <p className="text-sm">{product.stock ? "В наличии: " + product.stock : "Нет в наличии"}</p>
            {product.description && <p className="whitespace-pre-wrap">{product.description}</p>}
            {product.details && (
              <section className="space-y-1">
                <h2 className="font-medium">Характеристики</h2>
                <p className="whitespace-pre-wrap text-sm">{product.details}</p>
              </section>
            )}
            {product.variants && <p className="text-sm">Варианты: {product.variants}</p>}
          </div>
        </div>
      ) : (
        <h1 className="text-2xl font-semibold">Товар не найден</h1>
      )}
    </div>
  );
}
