import type { Metadata } from "next";
import { EditProduct } from "@/components/seller/product-form";

export const metadata: Metadata = { title: "Редактирование товара" };

export default async function EditProductPage({ params }: PageProps<"/seller/products/[id]/edit">) {
  const { id } = await params;
  return <EditProduct id={id} />;
}
