import type { Metadata } from "next";
import { ProductDetails } from "@/components/catalog/product-details";

export const metadata: Metadata = { title: "Страница товара" };

export default async function CatalogProductPage({ params }: PageProps<"/catalog/[id]">) {
  const { id } = await params;
  return <ProductDetails id={id} />;
}
