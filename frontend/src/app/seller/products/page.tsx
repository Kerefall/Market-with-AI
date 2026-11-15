import type { Metadata } from "next";
import { ProductList } from "@/components/seller/product-list";

export const metadata: Metadata = { title: "Товары" };

export default function ProductsPage() {
  return <ProductList />;
}
