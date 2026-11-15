import type { Metadata } from "next";
import { ProductForm } from "@/components/seller/product-form";

export const metadata: Metadata = { title: "Новый товар" };

export default function NewProductPage() {
  return <ProductForm />;
}
