import type { Metadata } from "next";
import { Catalog } from "@/components/catalog/catalog";

export const metadata: Metadata = { title: "Каталог" };

export default function CatalogPage() {
  return <Catalog />;
}
