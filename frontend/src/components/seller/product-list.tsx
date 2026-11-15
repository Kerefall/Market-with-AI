"use client";

import Link from "next/link";
import { Button } from "@/components/ui/button";
import { formatPrice, useDemoStore } from "@/lib/demo-store";

export function ProductList() {
  const { products } = useDemoStore();

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-2xl font-semibold">Товары</h1>
        <Button asChild><Link href="/seller/products/new">Новый товар</Link></Button>
      </div>
      {products.length === 0 ? (
        <p className="text-sm text-muted-foreground">Добавьте первый товар.</p>
      ) : (
        <div className="overflow-x-auto rounded-lg border">
          <table className="w-full text-left text-sm">
            <thead className="bg-muted font-medium">
              <tr>
                {["Название", "Категория", "Цена", "Остаток", "Статус", "Действия"].map((label) => (
                  <th key={label} scope="col" className="px-3 py-2 font-medium">{label}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {products.map((product) => (
                <tr key={product.id} className="border-t">
                  <th scope="row" className="px-3 py-2 font-medium">{product.name}</th>
                  <td className="px-3 py-2">{product.category}</td>
                  <td className="whitespace-nowrap px-3 py-2">{formatPrice(product.price)}</td>
                  <td className="px-3 py-2">{product.stock}</td>
                  <td className="whitespace-nowrap px-3 py-2">
                    {product.status === "published" ? "Опубликован" : "Черновик"}
                  </td>
                  <td className="px-3 py-2">
                    <Link
                      className="underline underline-offset-2"
                      href={"/seller/products/" + product.id + "/edit"}
                      aria-label={"Редактировать товар «" + product.name + "»"}
                    >
                      Редактировать
                    </Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
